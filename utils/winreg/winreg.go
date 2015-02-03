// +build windows,!linux

package winreg

import (
	"strings"
	"syscall"
	"unsafe"

	"github.com/juju/errors"
)

func resolveSubtreeFromString(subtree string) (result syscall.Handle, err error) {
	switch subtree {
	case "HKLM":
		result = syscall.HKEY_LOCAL_MACHINE
	case "HKCC":
		result = syscall.HKEY_CURRENT_CONFIG
	case "HKCR":
		result = syscall.HKEY_CLASSES_ROOT
	case "HKCU":
		result = syscall.HKEY_CURRENT_USER
	case "HKU":
		result = syscall.HKEY_USERS
	default:
		err = errors.Errorf("Unknown HKEY: %s", subtree)
	}
	return
}

func splitRegPath(path string) (subtree syscall.Handle, key string, err error) {
	splitPath := strings.Split(path, ":\\")
	if len(splitPath) != 2 {
		return 0, "", errors.Errorf("Registry path incorrect. Please use SUBTREE:\\path\\to\\key")
	}
	subtree, err = resolveSubtreeFromString(splitPath[0])
	if err != nil {
		return
	}
	key = splitPath[1]
	return
}

func ReadRegistryString(path, name string) (result string, err error) {
	subtree, key, err := splitRegPath(path)
	if err != nil {
		return
	}

	var typ uint32
	var buf [syscall.MAX_LONG_PATH]uint16
	var h syscall.Handle

	err = syscall.RegOpenKeyEx(subtree, syscall.StringToUTF16Ptr(key), 0, syscall.KEY_READ, &h)
	if err != nil {
		return
	}
	defer syscall.RegCloseKey(h)

	n := uint32(len(buf))
	err = syscall.RegQueryValueEx(h, syscall.StringToUTF16Ptr(name), nil, &typ, (*byte)(unsafe.Pointer(&buf[0])), &n)
	if err != nil {
		return
	}
	result = syscall.UTF16ToString(buf[:])
	return
}

func WriteRegistryString(path, name, value string) (err error) {
	var handle syscall.Handle
	subtree, key, err := splitRegPath(path)
	if err != nil {
		return
	}
	err = syscall.RegOpenKeyEx(subtree, syscall.StringToUTF16Ptr(""), 0, syscall.KEY_CREATE_SUB_KEY, &handle)
	if err != nil {
		return
	}

	defer syscall.RegCloseKey(handle)
	var d uint32
	err = RegCreateKeyEx(
		handle,
		syscall.StringToUTF16Ptr(key),
		0, nil, 0, syscall.KEY_ALL_ACCESS, nil, &handle, &d)
	if err != nil {
		return err
	}
	buf := syscall.StringToUTF16(value)
	err = RegSetValueEx(
		handle,
		syscall.StringToUTF16Ptr(name),
		0, syscall.REG_SZ, (*byte)(unsafe.Pointer(&buf[0])), uint32(len(buf)*2))

	if err != nil {
		return err
	}
	return nil
}

// DeleteRegistryKey will delete a windows registry key.
func DeleteRegistryKey(path, name string) (err error) {
	subtree, key, err := splitRegPath(path)
	var handle syscall.Handle

	err = syscall.RegOpenKeyEx(subtree, syscall.StringToUTF16Ptr(key), 0, syscall.KEY_READ, &handle)
	if err != nil {
		return
	}
	defer syscall.RegCloseKey(handle)
	err = RegDeleteKey(handle, syscall.StringToUTF16Ptr(name))
	return
}

// DeleteRegistryKeyValue will delete a windows registry value from the given key.
func DeleteRegistryKeyValue(path, name string) (err error) {
	subtree, key, err := splitRegPath(path)
	var handle syscall.Handle

	err = syscall.RegOpenKeyEx(subtree, syscall.StringToUTF16Ptr(key), 0, syscall.KEY_READ, &handle)
	if err != nil {
		return
	}
	defer syscall.RegCloseKey(handle)
	err = RegDeleteKeyValue(handle, syscall.StringToUTF16Ptr(""), syscall.StringToUTF16Ptr(name))
	return
}
