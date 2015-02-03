package cloudinit

var winPowershellHelperFunctions = `

$ErrorActionPreference = "Stop"

Import-Module Carbon

function ExecRetry($command, $maxRetryCount = 10, $retryInterval=2)
{
    $currErrorActionPreference = $ErrorActionPreference
    $ErrorActionPreference = "Continue"

    $retryCount = 0
    while ($true)
    {
        try
        {
            & $command
            break
        }
        catch [System.Exception]
        {
            $retryCount++
            if ($retryCount -ge $maxRetryCount)
            {
                $ErrorActionPreference = $currErrorActionPreference
                throw
            }
            else
            {
                Write-Error $_.Exception
                Start-Sleep $retryInterval
            }
        }
    }

    $ErrorActionPreference = $currErrorActionPreference
}

function create-account ([string]$accountName, [string]$accountDescription, [string]$password) {
	$hostname = hostname
	$comp = [adsi]"WinNT://$hostname"
	$user = $comp.Create("User", $accountName)
	$user.SetPassword($password)
	$user.SetInfo()
	$user.description = $accountDescription
	$user.SetInfo()
	$User.UserFlags[0] = $User.UserFlags[0] -bor 0x10000
	$user.SetInfo()

	$objOU = [ADSI]"WinNT://$hostname/Administrators,group"
	$objOU.add("WinNT://$hostname/$accountName")
}

$Source = @"
using System;
using System.Text;
using System.Runtime.InteropServices;

namespace PSCloudbase
{
    public sealed class Win32CryptApi
    {
        public static long CRYPT_SILENT                     = 0x00000040;
        public static long CRYPT_VERIFYCONTEXT              = 0xF0000000;
        public static int PROV_RSA_FULL                     = 1;

        [DllImport("advapi32.dll", CharSet=CharSet.Auto, SetLastError=true)]
        [return : MarshalAs(UnmanagedType.Bool)]
        public static extern bool CryptAcquireContext(ref IntPtr hProv,
                                                      StringBuilder pszContainer, // Don't use string, as Powershell replaces $null with an empty string
                                                      StringBuilder pszProvider, // Don't use string, as Powershell replaces $null with an empty string
                                                      uint dwProvType,
                                                      uint dwFlags);

        [DllImport("Advapi32.dll", EntryPoint = "CryptReleaseContext", CharSet = CharSet.Unicode, SetLastError = true)]
        public static extern bool CryptReleaseContext(IntPtr hProv, Int32 dwFlags);

        [DllImport("advapi32.dll", SetLastError=true)]
        public static extern bool CryptGenRandom(IntPtr hProv, uint dwLen, byte[] pbBuffer);

        [DllImport("Kernel32.dll")]
        public static extern uint GetLastError();
    }
}
"@

Add-Type -TypeDefinition $Source -Language CSharp

function Get-RandomPassword
{
    [CmdletBinding()]
    param
    (
        [parameter(Mandatory=$true)]
        [int]$Length
    )
    process
    {
        $hProvider = 0
        try
        {
            if(![PSCloudbase.Win32CryptApi]::CryptAcquireContext([ref]$hProvider, $null, $null,
                                                                 [PSCloudbase.Win32CryptApi]::PROV_RSA_FULL,
                                                                 ([PSCloudbase.Win32CryptApi]::CRYPT_VERIFYCONTEXT -bor
                                                                  [PSCloudbase.Win32CryptApi]::CRYPT_SILENT)))
            {
                throw "CryptAcquireContext failed with error: 0x" + "{0:X0}" -f [PSCloudbase.Win32CryptApi]::GetLastError()
            }

            $buffer = New-Object byte[] $Length
            if(![PSCloudbase.Win32CryptApi]::CryptGenRandom($hProvider, $Length, $buffer))
            {
                throw "CryptGenRandom failed with error: 0x" + "{0:X0}" -f [PSCloudbase.Win32CryptApi]::GetLastError()
            }

            $buffer | ForEach-Object { $password += "{0:X0}" -f $_ }
            return $password
        }
        finally
        {
            if($hProvider)
            {
                $retVal = [PSCloudbase.Win32CryptApi]::CryptReleaseContext($hProvider, 0)
            }
        }
    }
}

$Source = @"
using System;
using System.Text;
using System.Runtime.InteropServices;
using System.Security.Principal;
using System.ComponentModel;

namespace PSCloudbase
{
    public class ProcessManager
    {
        const int LOGON32_LOGON_SERVICE = 5;
        const int LOGON32_PROVIDER_DEFAULT = 0;
        const int TOKEN_ALL_ACCESS = 0x000f01ff;
        const uint GENERIC_ALL_ACCESS = 0x10000000;
        const uint INFINITE = 0xFFFFFFFF;
        const uint PI_NOUI = 0x00000001;
        const uint WAIT_FAILED = 0xFFFFFFFF;

        enum SECURITY_IMPERSONATION_LEVEL
        {
            SecurityAnonymous,
            SecurityIdentification,
            SecurityImpersonation,
            SecurityDelegation
        }

        enum TOKEN_TYPE
        {
            TokenPrimary = 1,
            TokenImpersonation
        }

        [StructLayout(LayoutKind.Sequential)]
        struct SECURITY_ATTRIBUTES
        {
            public int nLength;
            public IntPtr lpSecurityDescriptor;
            public int bInheritHandle;
        }

        [StructLayout(LayoutKind.Sequential)]
        struct PROCESS_INFORMATION
        {
            public IntPtr hProcess;
            public IntPtr hThread;
            public int dwProcessId;
            public int dwThreadId;
        }

        [StructLayout(LayoutKind.Sequential, CharSet = CharSet.Unicode)]
        struct STARTUPINFO
        {
            public Int32 cb;
            public string lpReserved;
            public string lpDesktop;
            public string lpTitle;
            public Int32 dwX;
            public Int32 dwY;
            public Int32 dwXSize;
            public Int32 dwYSize;
            public Int32 dwXCountChars;
            public Int32 dwYCountChars;
            public Int32 dwFillAttribute;
            public Int32 dwFlags;
            public Int16 wShowWindow;
            public Int16 cbReserved2;
            public IntPtr lpReserved2;
            public IntPtr hStdInput;
            public IntPtr hStdOutput;
            public IntPtr hStdError;
        }

        [StructLayout(LayoutKind.Sequential)]
        struct PROFILEINFO {
            public int dwSize;
            public uint dwFlags;
            [MarshalAs(UnmanagedType.LPTStr)]
            public String lpUserName;
            [MarshalAs(UnmanagedType.LPTStr)]
            public String lpProfilePath;
            [MarshalAs(UnmanagedType.LPTStr)]
            public String lpDefaultPath;
            [MarshalAs(UnmanagedType.LPTStr)]
            public String lpServerName;
            [MarshalAs(UnmanagedType.LPTStr)]
            public String lpPolicyPath;
            public IntPtr hProfile;
        }

        [StructLayout(LayoutKind.Sequential, CharSet = CharSet.Unicode)]
        public struct USER_INFO_4
        {
            public string name;
            public string password;
            public int password_age;
            public uint priv;
            public string home_dir;
            public string comment;
            public uint flags;
            public string script_path;
            public uint auth_flags;
            public string full_name;
            public string usr_comment;
            public string parms;
            public string workstations;
            public int last_logon;
            public int last_logoff;
            public int acct_expires;
            public int max_storage;
            public int units_per_week;
            public IntPtr logon_hours;    // This is a PBYTE
            public int bad_pw_count;
            public int num_logons;
            public string logon_server;
            public int country_code;
            public int code_page;
            public IntPtr user_sid;     // This is a PSID
            public int primary_group_id;
            public string profile;
            public string home_dir_drive;
            public int password_expired;
        }

        [DllImport("advapi32.dll", CharSet=CharSet.Auto, SetLastError=true)]
        extern static bool DuplicateTokenEx(
            IntPtr hExistingToken,
            uint dwDesiredAccess,
            ref SECURITY_ATTRIBUTES lpTokenAttributes,
            SECURITY_IMPERSONATION_LEVEL ImpersonationLevel,
            TOKEN_TYPE TokenType,
            out IntPtr phNewToken);

        [DllImport("advapi32.dll", SetLastError=true)]
        static extern bool LogonUser(
            string lpszUsername,
            string lpszDomain,
            string lpszPassword,
            int dwLogonType,
            int dwLogonProvider,
            out IntPtr phToken);

        [DllImport("advapi32.dll", SetLastError=true, CharSet=CharSet.Auto)]
        static extern bool CreateProcessAsUser(
            IntPtr hToken,
            string lpApplicationName,
            string lpCommandLine,
            ref SECURITY_ATTRIBUTES lpProcessAttributes,
            ref SECURITY_ATTRIBUTES lpThreadAttributes,
            bool bInheritHandles,
            uint dwCreationFlags,
            IntPtr lpEnvironment,
            string lpCurrentDirectory,
            ref STARTUPINFO lpStartupInfo,
            out PROCESS_INFORMATION lpProcessInformation);

        [DllImport("kernel32.dll", SetLastError=true)]
        static extern UInt32 WaitForSingleObject(IntPtr hHandle,
                                                 UInt32 dwMilliseconds);

        [DllImport("Kernel32.dll")]
        static extern int GetLastError();

        [DllImport("Kernel32.dll")]
        extern static int CloseHandle(IntPtr handle);

        [DllImport("kernel32.dll", SetLastError = true)]
        [return: MarshalAs(UnmanagedType.Bool)]
        static extern bool GetExitCodeProcess(IntPtr hProcess,
                                              out uint lpExitCode);

        [DllImport("userenv.dll", SetLastError=true, CharSet=CharSet.Auto)]
        [return: MarshalAs(UnmanagedType.Bool)]
        static extern bool LoadUserProfile(IntPtr hToken,
                                           ref PROFILEINFO lpProfileInfo);

        [DllImport("userenv.dll", SetLastError=true, CharSet=CharSet.Auto)]
        [return: MarshalAs(UnmanagedType.Bool)]
        static extern bool UnloadUserProfile(IntPtr hToken, IntPtr hProfile);

         [DllImport("Netapi32.dll", CharSet=CharSet.Unicode, ExactSpelling=true)]
        extern static int NetUserGetInfo(
            [MarshalAs(UnmanagedType.LPWStr)] string ServerName,
            [MarshalAs(UnmanagedType.LPWStr)] string UserName,
            int level, out IntPtr BufPtr);

        public static uint RunProcess(string userName, string password,
                                      string domain, string cmd,
                                      string arguments,
                                      bool loadUserProfile = true)
        {
            bool retValue;
            IntPtr phToken = IntPtr.Zero;
            IntPtr phTokenDup = IntPtr.Zero;
            PROCESS_INFORMATION pInfo = new PROCESS_INFORMATION();
            PROFILEINFO pi = new PROFILEINFO();

            try
            {
                retValue = LogonUser(userName, domain, password,
                                     LOGON32_LOGON_SERVICE,
                                     LOGON32_PROVIDER_DEFAULT,
                                     out phToken);
                if(!retValue)
                    throw new Win32Exception(GetLastError());

                var sa = new SECURITY_ATTRIBUTES();
                sa.nLength = Marshal.SizeOf(sa);

                retValue = DuplicateTokenEx(
                    phToken, GENERIC_ALL_ACCESS, ref sa,
                    SECURITY_IMPERSONATION_LEVEL.SecurityImpersonation,
                    TOKEN_TYPE.TokenPrimary, out phTokenDup);
                if(!retValue)
                    throw new Win32Exception(GetLastError());

                STARTUPINFO sInfo = new STARTUPINFO();
                sInfo.lpDesktop = "";

                if(loadUserProfile)
                {
                    IntPtr userInfoPtr = IntPtr.Zero;
                    int retValueNetUser = NetUserGetInfo(null, userName, 4,
                                                         out userInfoPtr);
                    if(retValueNetUser != 0)
                        throw new Win32Exception(retValueNetUser);

                    USER_INFO_4 userInfo = (USER_INFO_4)Marshal.PtrToStructure(
                        userInfoPtr, typeof(USER_INFO_4));

                    pi.dwSize = Marshal.SizeOf(pi);
                    pi.dwFlags = PI_NOUI;
                    pi.lpUserName = userName;
                    pi.lpProfilePath = userInfo.profile;

                    retValue = LoadUserProfile(phTokenDup, ref pi);
                    if(!retValue)
                        throw new Win32Exception(GetLastError());
                }

                retValue = CreateProcessAsUser(phTokenDup, cmd, arguments,
                                               ref sa, ref sa, false, 0,
                                               IntPtr.Zero, null,
                                               ref sInfo, out pInfo);
                if(!retValue)
                    throw new Win32Exception(GetLastError());

                if(WaitForSingleObject(pInfo.hProcess, INFINITE) == WAIT_FAILED)
                    throw new Win32Exception(GetLastError());

                uint exitCode;
                retValue = GetExitCodeProcess(pInfo.hProcess, out exitCode);
                if(!retValue)
                    throw new Win32Exception(GetLastError());

                return exitCode;
            }
            finally
            {
                if(pi.hProfile != IntPtr.Zero)
                    UnloadUserProfile(phTokenDup, pi.hProfile);
                if(phToken != IntPtr.Zero)
                    CloseHandle(phToken);
                if(phTokenDup != IntPtr.Zero)
                    CloseHandle(phTokenDup);
                if(pInfo.hProcess != IntPtr.Zero)
                    CloseHandle(pInfo.hProcess);
            }
        }
    }
}
"@

Add-Type -TypeDefinition $Source -Language CSharp

function Start-ProcessAsUser
{
    [CmdletBinding()]
    param
    (
        [parameter(Mandatory=$true, ValueFromPipeline=$true)]
        [string]$Command,

        [parameter()]
        [string]$Arguments,

        [parameter(Mandatory=$true)]
        [PSCredential]$Credential,

        [parameter()]
        [bool]$LoadUserProfile = $true
    )
    process
    {
        $nc = $Credential.GetNetworkCredential()

        $domain = "."
        if($nc.Domain)
        {
            $domain = $nc.Domain
        }

        [PSCloudbase.ProcessManager]::RunProcess($nc.UserName, $nc.Password,
                                                 $domain, $Command,
                                                 $Arguments, $LoadUserProfile)
    }
}

$powershell = "$ENV:SystemRoot\System32\WindowsPowerShell\v1.0\powershell.exe"
$cmdExe = "$ENV:SystemRoot\System32\cmd.exe"

$juju_passwd = Get-RandomPassword 20
$juju_passwd += "^"
create-account jujud "Juju Admin user" $juju_passwd
$hostname = hostname
$juju_user = "$hostname\jujud"

Grant-Privilege $juju_user SeServiceLogonRight
Grant-Privilege $juju_user SeAssignPrimaryTokenPrivilege

New-ItemProperty "HKLM:\Software\Microsoft\Windows NT\CurrentVersion\Winlogon\SpecialAccounts\UserList" -Name "jujud" -Value 0 -PropertyType "DWord"

$secpasswd = ConvertTo-SecureString $juju_passwd -AsPlainText -Force
$jujuCreds = New-Object System.Management.Automation.PSCredential ($juju_user, $secpasswd)

$jujudRegKey = Get-Item "HKLM:\SOFTWARE\Wow6432Node\jujud" -ErrorAction SilentlyContinue

if (!$jujudRegKey){
    $jujudRegKey = New-Item "HKLM:\SOFTWARE\Wow6432Node" -Name "jujud"
}

$secPasswdAsString = $secpasswd | ConvertFrom-SecureString

New-ItemProperty "HKLM:\Software\Wow6432Node\juju" -Name "Password" -PropertyType "string" -Value $secPasswdAsString

`
