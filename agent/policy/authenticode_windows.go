//go:build windows

package policy

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	certQueryObjectFile                   = 1
	certQueryContentFlagPKCS7SignedEmbed  = 0x00000400
	certQueryContentFlagPKCS7SignedEmbed2 = 0x00000800
	certQueryContentFlagCatalog           = 0x00000040 // CERT_QUERY_CONTENT_FLAG_CATALOG
	certQueryFormatFlagBinary             = 2
	certQueryContentFlagAllTypes          = 0xffffffff
	certQueryFormatFlagAllTypes           = 0xffffffff
)

// filePublisher returns a display name for the Authenticode signer (CERT_NAME_SIMPLE_DISPLAY_TYPE).
// Uses CryptQueryObject + first certificate in the returned store (typical for embedded PKCS#7).
func filePublisher(path string) (string, error) {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}

	var lastErr error
	for _, contentFlags := range []uint32{
		certQueryContentFlagPKCS7SignedEmbed,
		certQueryContentFlagPKCS7SignedEmbed2,
		certQueryContentFlagPKCS7SignedEmbed | certQueryContentFlagPKCS7SignedEmbed2,
		certQueryContentFlagCatalog,
		certQueryContentFlagCatalog | certQueryContentFlagPKCS7SignedEmbed,
		certQueryContentFlagAllTypes,
	} {
		for _, formatFlags := range []uint32{certQueryFormatFlagBinary, certQueryFormatFlagAllTypes} {
			pub, err := cryptQueryPublisher(pathPtr, contentFlags, formatFlags)
			if err == nil && pub != "" {
				return pub, nil
			}
			lastErr = err
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("CryptQueryObject failed")
	}

	// Catalog-signed OS binaries: WinTrust file verify may not load catalog; WTHelper path can miss.
	pub, werr := publisherFromWinTrust(path)
	if werr == nil {
		return pub, nil
	}
	psPub, psErr := publisherFromPowerShell(path)
	if psErr == nil && psPub != "" {
		return psPub, nil
	}
	if lastErr != nil {
		return "", fmt.Errorf("%w; wintrust: %v; powershell: %v", lastErr, werr, psErr)
	}
	return "", fmt.Errorf("wintrust: %v; powershell: %v", werr, psErr)
}

func cryptQueryPublisher(pathPtr *uint16, contentFlags, formatFlags uint32) (string, error) {
	var (
		encodingType, contentType, formatType uint32
		certStore                           windows.Handle
		cryptMsg                            windows.Handle
		ctx                                 uintptr
	)

	r, _, _ := procCryptQueryObject.Call(
		uintptr(certQueryObjectFile),
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(contentFlags),
		uintptr(formatFlags),
		0,
		uintptr(unsafe.Pointer(&encodingType)),
		uintptr(unsafe.Pointer(&contentType)),
		uintptr(unsafe.Pointer(&formatType)),
		uintptr(unsafe.Pointer(&certStore)),
		uintptr(unsafe.Pointer(&cryptMsg)),
		uintptr(unsafe.Pointer(&ctx)),
	)
	if r == 0 {
		return "", fmt.Errorf("CryptQueryObject failed")
	}
	defer procCertCloseStore.Call(uintptr(certStore), 0)
	if cryptMsg != 0 {
		defer procCryptMsgClose.Call(uintptr(cryptMsg))
	}
	if ctx != 0 {
		defer procCertFreeCertificateContext.Call(ctx)
	}

	// Prefer explicit context from CryptQueryObject when present.
	if ctx != 0 {
		name, nerr := certContextDisplayName(ctx)
		if nerr == nil && name != "" {
			return name, nil
		}
	}

	var p uintptr
	for {
		next, _, _ := procCertEnumCertificatesInStore.Call(uintptr(certStore), p)
		if next == 0 {
			break
		}
		name, nerr := certContextDisplayName(next)
		if nerr == nil && name != "" {
			procCertFreeCertificateContext.Call(next)
			return name, nil
		}
		p = next
	}
	return "", fmt.Errorf("no certificate with display name")
}

func certContextDisplayName(pCertContext uintptr) (string, error) {
	var size uint32
	r, _, _ := procCertGetNameStringW.Call(
		pCertContext,
		uintptr(certNameSimpleDisplayType),
		0,
		0,
		0,
		uintptr(unsafe.Pointer(&size)),
	)
	if r == 0 || size <= 1 {
		return "", fmt.Errorf("CertGetNameStringW size failed")
	}
	buf := make([]uint16, size)
	r, _, _ = procCertGetNameStringW.Call(
		pCertContext,
		uintptr(certNameSimpleDisplayType),
		0,
		0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if r == 0 {
		return "", fmt.Errorf("CertGetNameStringW failed")
	}
	return syscall.UTF16ToString(buf), nil
}

const certNameSimpleDisplayType = 4

var (
	modcrypt32 = windows.NewLazySystemDLL("crypt32.dll")

	procCryptQueryObject            = modcrypt32.NewProc("CryptQueryObject")
	procCertCloseStore              = modcrypt32.NewProc("CertCloseStore")
	procCertEnumCertificatesInStore = modcrypt32.NewProc("CertEnumCertificatesInStore")
	procCertFreeCertificateContext  = modcrypt32.NewProc("CertFreeCertificateContext")
	procCertGetNameStringW          = modcrypt32.NewProc("CertGetNameStringW")
	procCryptMsgClose               = modcrypt32.NewProc("CryptMsgClose")
)
