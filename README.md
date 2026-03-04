# ctfextension - Chrome to Firefox Extension Converter

A command-line tool written in GO that converts Chrome browser extensions to Firefox-compatible format. Accepts both unpacked extension folders and .crx files.

---

## Installation

ctfextension is built from source. You compile it once on your machine and it is ready to use.

All platforms require Git and Go 1.22 or later.

---

### Linux

**1. Install Go**

```bash
# Ubuntu / Debian
sudo apt install golang-go

# Fedora / RHEL
sudo dnf install golang

# Arch
sudo pacman -S go
```

Verify: `go version`

**2. Install Git (if not already installed)**

```bash
# Ubuntu / Debian
sudo apt install git

# Fedora / RHEL
sudo dnf install git

# Arch
sudo pacman -S git
```

**3. Clone and build**

```bash
git clone https://github.com/neo-meta-cortex/chrome-to-firefox-converter.git
cd chrome-to-firefox-converter
go build -o ctfextension ./cmd/main.go
```

**4. Make it available system-wide**

```bash
sudo mv ctfextension /usr/local/bin/ctfextension
```

You can now run `ctfextension` from any directory.

---

### macOS

**1. Install Go**

The easiest way is via [Homebrew](https://brew.sh):

```bash
brew install go
```

Or download the installer from [go.dev/dl](https://go.dev/dl/) and run the `.pkg` file.

Verify: `go version`

**2. Install Git (if not already installed)**

```bash
brew install git
```

Or install Xcode Command Line Tools which includes Git:

```bash
xcode-select --install
```

**3. Clone and build**

```bash
git clone https://github.com/neo-meta-cortex/chrome-to-firefox-converter.git
cd chrome-to-firefox-converter
go build -o ctfextension ./cmd/main.go
```

**4. Make it available system-wide**

```bash
sudo mv ctfextension /usr/local/bin/ctfextension
```

> **Apple Silicon (M1/M2/M3):** The build command above works natively. Go detects your architecture automatically.

You can now run `ctfextension` from any terminal.

---

### Windows

**1. Install Go**

Download and run the Windows installer from [go.dev/dl](https://go.dev/dl/).
The installer adds Go to your PATH automatically.

Verify by opening Command Prompt or PowerShell and running:

```
go version
```

**2. Install Git**

Download and install from [git-scm.com](https://git-scm.com/download/win).
During setup, choose **"Git from the command line and also from 3rd-party software"** when prompted.

**3. Clone and build**

Open Command Prompt or PowerShell:

```powershell
git clone https://github.com/neo-meta-cortex/chrome-to-firefox-converter.git
cd chrome-to-firefox-converter
go build -o ctfextension.exe ./cmd/main.go
```

**4. Make it available system-wide**

Move `ctfextension.exe` to a folder that is already on your PATH:

```powershell
move ctfextension.exe C:\Windows\System32\ctfextension.exe
```

Or add the project folder to your PATH manually:

1. Search for **"Environment Variables"** in the Start menu
2. Click **"Edit the system environment variables"**
3. Click **Environment Variables**, find **Path**, click **Edit**
4. Click **New** and add the full path to the folder containing `ctfextension.exe`

You can now run `ctfextension` from any Command Prompt or PowerShell window.


---
# How to Get a Chrome Extension for Conversion

There are two ways to get a Chrome extension ready to convert.

---

## Option A - Install in Chrome

The easiest method. ctfextension accepts .crx files directly.

1. Install the extension in Chrome normally from the [Chrome Web Store](https://chromewebstore.google.com)
2. Open a new tab and go to `chrome://extensions`
3. Enable **Developer mode** using the toggle in the top right corner
4. Find the extension you want and note its ID (a long string like `aapbdbdomjkkjkaonfhkkikfgjllcleb`)
5. The unpacked .crx file is stored on your machine at(Each extension folder contains a version subfolder (e.g. `1.0.0_0`). The files inside that folder are the unpacked extension — you can pass that folder directly to ctfextension.):

**Windows**
```
C:\Users\<YourName>\AppData\Local\Google\Chrome\User Data\Default\Extensions\<extension-id>\
```

**macOS**
```
~/Library/Application Support/Google/Chrome/Default/Extensions/<extension-id>/
```

**Linux**
```
~/.config/google-chrome/Default/Extensions/<extension-id>/
```


---

## Option B - Download and unpack a .crx manually

If you want the raw .crx file rather than the already-extracted folder:

1. Go to [crxextractor.com](https://crxextractor.com) or [chrome-stats.com](https://chrome-stats.com)
2. Paste the Chrome Web Store URL of the extension
3. Download the .crx file
4. Pass it directly to ctfextension — it will extract it automatically

---

## Finding the extension ID

The extension ID is visible on the `chrome://extensions` page when Developer mode is enabled. It appears underneath the extension name as a 32-character string, for example:

```
aapbdbdomjkkjkaonfhkkikfgjllcleb
```

Use this to locate the extension folder in the paths listed in Option A above.


## Usage

```bash
ctfextension <input-path> <output-path> [--xpi]
```

| Argument | Description |
|---|---|
| `input-path` | Path to a `.crx` file or an unpacked Chrome extension folder |
| `output-path` | Where to write the converted Firefox extension |
| `--xpi` | Also package the output as a `.xpi` file for Firefox |

**Examples:**

```bash
# Convert a downloaded .crx file
ctfextension extension.crx ./my-firefox-ext

# Convert an unpacked extension folder
ctfextension ./my-chrome-ext ./my-firefox-ext

# Convert and produce a .xpi ready to install in Firefox
ctfextension extension.crx ./my-firefox-ext --xpi
```

Run `ctfextension -h` for the full help menu.

---

## What gets converted

**Manifest changes**

| Chrome | Firefox |
|---|---|
| `manifest_version: 3` | `manifest_version: 2` |
| `action` | `browser_action` |
| `background.service_worker` | `background.scripts` array |
| `host_permissions` | merged into `permissions` |
| MV3 `web_accessible_resources` | MV2 format |
| MV3 `content_security_policy` | MV2 string format |
| `.crx` archive | extracted automatically before conversion |

**Permissions stripped automatically**

These Chrome-only permissions have no Firefox equivalent and are removed from the output manifest:

| Permission | Reason |
|---|---|
| `sidePanel` | Chrome-only sidebar API |
| `debugger` | Chrome-only DevTools protocol |
| `offscreen` | Chrome MV3 only |
| `system.display` | Chrome-only display management |
| `tabCapture` | Chrome-only |
| `pageCapture` | Chrome-only |
| `audio` | Chrome-only |
| `transientBackground` | Chrome MV3 only |

**Manifest fields removed automatically**

These Chrome-only fields cause Firefox to reject the extension and are removed:

| Field | Reason |
|---|---|
| `key` | Chrome extension signing key |
| `update_url` | Chrome Web Store update mechanism |
| `externally_connectable` | Chrome-only cross-extension messaging |

**JavaScript**

All `.js`, `.mjs`, and `.ts` files have `chrome.*` API calls renamed to `browser.*`.

**Other files**

All other files including HTML, CSS, images, and JSON are copied to the output unchanged.

---

## Testing your converted extension in Firefox

1. Open Firefox and navigate to `about:debugging`
2. Click **This Firefox**
3. Click **Load Temporary Add-on**
4. Select any file inside your output folder

---

## Limitations

Some Chrome APIs have no Firefox equivalent. The tool will warn you if it finds them:

- `chrome.enterprise.*`
- `chrome.declarativeNetRequest` (partial support only)
- `chrome.certificateProvider`
- `chrome.fileBrowserHandler`
- `chrome.loginState`
- `chrome.platformKeys`

---

## License

MIT
