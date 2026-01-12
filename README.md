<h2 align="center">ZIMServer</h2>
<p align="center">A modern and lightweight alternative to kiwix-serve for your ZIM files</p>
<p align="center">
    <a href="#about">About</a> •
    <a href="#why-zimserver">Why ZIMServer</a> •
    <a href="#use-cases">Use Cases</a> •
    <a href="#installation">Installation</a> •
    <a href="#usage">Usage</a> •
    <a href="#license">License</a>
</p>

## About

ZIMServer lets you serve ZIM files (Wikipedia, Wiktionary, etc.) through a clean Web interface. It's basically kiwix-serve but simpler to use and with a more modern look.

## Why ZIMServer

### Single binary, no dependencies
Unlike kiwix-serve which needs system libraries, ZIMServer is just one file. Download it, run it, that's it. No dependencies, no headaches.

### Zero configuration
Point it at your ZIM files and go. No config files to write, no settings to tweak. Want to serve files? `zimserver /path/to/zims`. Done.

### Modern interface
Clean UI that works on phones and desktops. Fast search when available, proper mobile support, all the basics you'd expect from a modern web app.

### Hot reload
Drop a new ZIM file in your folder and ZIMServer picks it up automatically. No need to restart anything. It even waits for files to finish copying before loading them.

### Actually lightweight
Runs fine on a Raspberry Pi. Won't eat your RAM or max out your CPU. Good for everything from old hardware to proper servers.

## Use Cases

### Offline Education
Schools and libraries without internet can host Wikipedia and educational materials locally. Students get access to millions of articles without connectivity.

### Censorship Resistance
In places where internet access is restricted or monitored, ZIMServer provides uncensored access to knowledge on local networks or USB drives.

### Emergency Response
Medical teams and disaster response units can carry WikiMed, first aid guides, and technical manuals in the field - all available offline.

### Remote Work
Researchers and engineers working in isolated locations can access reference materials without depending on satellite internet.

### Personal Knowledge Base
Keep your own offline library at home. Wikipedia snapshots, technical documentation, Project Gutenberg books - always available, always fast.

### Air-Gapped Networks
Deploy in secure facilities or classified environments where external connectivity is prohibited but knowledge access is essential.

## Installation

Grab a binary (Linux (amd64, i386, arm64, armv7), macOS (Intel/ARM), Windows (amd64, i386, arm64)) from [releases](https://github.com/gaetanlhf/ZIMServer/releases) or build from source.

### Build from source

Needs Go 1.24+.

```bash
git clone https://github.com/gaetanlhf/ZIMServer.git
cd ZIMServer
make build
```

## Usage

```bash
# Point it at your ZIM files
zimserver /path/to/zim-files

# Or specific files
zimserver wikipedia.zim wiktionary.zim

# Mix files and directories
zimserver file.zim /another/directory

# Serve on your network
zimserver --host 0.0.0.0 --port 8080 /path/to/zims
```

Open `http://localhost:8080` in your browser. That's it.

## ZIM files
Download from [library.kiwix.org](https://library.kiwix.org) - Wikipedia, Wiktionary, medical references, Stack Exchange, TED, books, and more.

## License

This program is free software: you can redistribute it and/or modify it under the terms of the GNU Affero General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License along with this program. If not, see http://www.gnu.org/licenses/.