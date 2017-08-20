# Flying Carpet
Wireless, encrypted file transfer over automatically configured ad hoc networking. No network infrastructure required (access point, router, switch). Just two laptops (Mac and/or Windows) with wireless chips in close range.

Don't have a flash drive? Don't have access to a wireless network or don't trust one? Need to move a file larger than 2GB between Mac and Windows but don't want to set up a file share? Try it out!

# Sample Usage
**On receiving end (Mac):**

`./flyingcarpet -receive transferred_movie.avi -peer windows`

*\[Write down password\]*

**On sending end (Windows):**

`flyingcarpet.exe -send movie.avi -peer mac`

*\[Enter password from Mac\]*

# Features:
+ Cross-platform, Mac and Windows.

+ Speeds over 120mbps (with laptops close together).

+ Does not use Bluetooth or your local network, just wireless chip to wireless chip.

+ Files encrypted in transit.

+ Large files supported (<10MB RAM usage while transferring a 4.5GB file).

+ Standalone binary, no installation required.

# Compilation instructions:
(Ready-to-use x64 binaries in `/bin` as well, must run `chmod u+x flyingcarpet` for Mac.)

`cd flyingcarpet`

`go get ./...`

`go build`

# Restrictions:
+ Disables your wireless internet connection while in use (does not apply to Windows when receiving)

+ On Mac: May have to click Allow or enter username and password at prompt to join ad-hoc network. (Clicking cancel may still work.)

+ On Windows: Must run as administrator to receive files (to allow connection through firewall and clear ARP cache). Right-click Command Prompt icon in Start menu and select "Run as administrator," or press Win+X, A. 

+ Windows laptop must support hosted networking. To find out if yours does, run `netsh wlan show drivers`. If the `Hosted network supported : ` line says `No`, you can't use this product. Known issue on Surface Pro 3 and later.

+ If you choose to receive a filename that is already present in your current directory, it will be overwritten.

+ After a successful transfer, Flying Carpet will attempt to rejoin you to your previous wireless networks. If there is an error midway through the process, this may fail.

Disclaimer: I am not a cryptography expert. This is a usable product in its current state, but is also an experiment and a work in progress. Do not use for private files if you think a skilled attacker is less than 100 feet from you and trying to intercept them.

Licenses for third-party tools and libraries used can be found in the "3rd_party_licenses" folder.

# Screenshots

![](pictures/macDemo.png)

![](pictures/winDemo.png)

# Planned features

+ Ubuntu support

+ GUI

+ On Windows, add support for WiFi Direct API so that chips/drivers without support for ad hoc (IBSS) hosted networks can use Flying Carpet (applies to Surface 3 Pro and later and some other newer Windows 10 PCs).

I will not have much time to work on this until mid-October, so it may be until November that progress is made on these items. If you've used Flying Carpet, please tell me whether it worked or not. I have had very little testing and feedback so far. Thank you for your interest!
