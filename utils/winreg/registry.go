// +build windows,!linux

package winreg

//sys   RegCreateKeyEx(key syscall.Handle, subkey *uint16, reserved uint32, class *uint16, options uint32, desired uint32, sa *syscall.SecurityAttributes, result *syscall.Handle, disposition *uint32) (regerrno error) = advapi32.RegCreateKeyExW
//sys   RegDeleteKey(key syscall.Handle, subkey *uint16) (regerrno error) = advapi32.RegDeleteKeyW
//sys   RegSetValueEx(key syscall.Handle, valueName *uint16, reserved uint32, vtype uint32, buf *byte, bufsize uint32) (regerrno error) = advapi32.RegSetValueExW
//sys   RegDeleteKeyValue(key syscall.Handle, subkey *uint16, valueName *uint16) (regerrno error) = advapi32.RegDeleteKeyValueW
