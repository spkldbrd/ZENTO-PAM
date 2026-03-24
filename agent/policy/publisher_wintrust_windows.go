//go:build windows

package policy

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modWinTrust                        = windows.NewLazySystemDLL("wintrust.dll")
	procWTHelperProvDataFromStateData  = modWinTrust.NewProc("WTHelperProvDataFromStateData")
	procWTHelperGetProvSignerFromChain = modWinTrust.NewProc("WTHelperGetProvSignerFromChain")
	procWTHelperGetProvCertFromChain   = modWinTrust.NewProc("WTHelperGetProvCertFromChain")
)

// publisherFromWinTrust resolves the signing certificate display name after WinVerifyTrustEx,
// including catalog-signed binaries (e.g. many files under System32).
func publisherFromWinTrust(path string) (string, error) {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}

	fi := windows.WinTrustFileInfo{
		Size:     uint32(unsafe.Sizeof(windows.WinTrustFileInfo{})),
		FilePath: pathPtr,
	}
	data := &windows.WinTrustData{
		Size:                            uint32(unsafe.Sizeof(windows.WinTrustData{})),
		UIChoice:                        windows.WTD_UI_NONE,
		RevocationChecks:                windows.WTD_REVOKE_NONE,
		UnionChoice:                     windows.WTD_CHOICE_FILE,
		FileOrCatalogOrBlobOrSgnrOrCert: unsafe.Pointer(&fi),
		StateAction: windows.WTD_STATEACTION_VERIFY,
		// Avoid WTD_SAFER_FLAG: it can block catalog-backed OS signatures (e.g. System32 binaries).
		ProvFlags: 0,
	}

	// HWND must be NULL (0); InvalidHWND (-1) breaks catalog-backed verification for some OS binaries.
	verifyErr := windows.WinVerifyTrustEx(windows.HWND(0), &windows.WINTRUST_ACTION_GENERIC_VERIFY_V2, data)
	if verifyErr != nil {
		data.StateAction = windows.WTD_STATEACTION_CLOSE
		_ = windows.WinVerifyTrustEx(windows.HWND(0), &windows.WINTRUST_ACTION_GENERIC_VERIFY_V2, data)
		return "", fmt.Errorf("WinVerifyTrustEx: %w", verifyErr)
	}

	defer func() {
		data.StateAction = windows.WTD_STATEACTION_CLOSE
		_ = windows.WinVerifyTrustEx(windows.HWND(0), &windows.WINTRUST_ACTION_GENERIC_VERIFY_V2, data)
	}()

	prov, _, _ := procWTHelperProvDataFromStateData.Call(uintptr(data.StateData))
	if prov == 0 {
		return "", fmt.Errorf("WTHelperProvDataFromStateData")
	}

	sgnr, _, _ := procWTHelperGetProvSignerFromChain.Call(prov, 0, 0, 0)
	if sgnr == 0 {
		return "", fmt.Errorf("WTHelperGetProvSignerFromChain")
	}

	pCertInfo, _, _ := procWTHelperGetProvCertFromChain.Call(sgnr, 0)
	if pCertInfo == 0 {
		return "", fmt.Errorf("WTHelperGetProvCertFromChain")
	}

	// CRYPT_PROVIDER_CERT { cbStruct, pCert, ... } — x64: pCert at offset 8 after cbStruct+padding.
	type cryptProviderCert struct {
		CbStruct uint32
		_        uint32
		PCert    uintptr
	}
	cpc := (*cryptProviderCert)(unsafe.Pointer(pCertInfo))
	if cpc.PCert == 0 {
		return "", fmt.Errorf("signer cert context nil")
	}
	return certContextDisplayName(cpc.PCert)
}
