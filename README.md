<h2 align="center">ZIMServer</h2>
<p align="center">A modern and lightweight alternative to kiwix-serve for your ZIM files</p>
<p align="center">
    <a href="#about">About</a> •
    <a href="#features">Features</a> •
    <a href="#use-cases">Use Cases</a> •
    <a href="#installation">Installation</a> •
    <a href="#usage">Usage</a> •
    <a href="#license">License</a>
</p>

## About

ZIMServer lets you serve ZIM files (Wikipedia, Wiktionary, etc.) through a clean Web interface.
It is a streamlined alternative to kiwix-serve with a focus on usability and modern web standards.

## Features

- **Single binary**: Statically compiled for every OS/architecture with no dependencies.
- **Zero configuration**: Point it at your ZIM files and start serving immediately.
- **Multi-archive support**: Serve an unlimited number of ZIM files simultaneously from folders or specific paths.
- **Hot reload**: Automatically detects and loads new ZIM files without needing a restart.
- **Fast search**: Optimized search functionality to find articles quickly across your library.
- **Responsive interface**: A clean, modern UI designed for both mobile and desktop browsers.
- **Multilingual support**: Native support with automatic detection and manual selection of the interface language.
- **Light & Dark themes**: Automatic system detection or manual toggle, with a forced dark mode option for ZIM content.
- **Resource efficient**: Low memory and CPU footprint, suitable for hardware ranging from Raspberry Pi to dedicated servers.

## Use Cases

- **Offline Education**: Host Wikipedia and educational materials locally for schools and libraries without internet access.
- **Censorship Resistance**: Provide uncensored access to knowledge in restricted regions via local networks or USB drives.
- **Emergency Response**: Carry WikiMed and technical manuals in the field for medical teams and disaster units.
- **Remote Work**: Access reference materials in isolated locations without depending on satellite connectivity.
- **Personal Knowledge Base**: Keep a fast, offline home library of Wikipedia, documentation, and books.
- **Air-Gapped Networks**: Deploy in secure or classified environments where external internet is prohibited.

## Installation

Grab a binary (Linux (amd64, i386, arm64, armv7), macOS (Intel/ARM), Windows (amd64, i386, arm64)) from [releases](https://github.com/gaetanlhf/ZIMServer/releases) or build from source.

### Build from source

Needs Go 1.26+.

```bash
git clone https://github.com/gaetanlhf/ZIMServer.git
cd ZIMServer
make build
```

## Usage

```bash
# Point it at your ZIM files
./zimserver /path/to/zim-files

# Or specific files
./zimserver wikipedia.zim wiktionary.zim

# Mix files and directories
./zimserver file.zim /another/directory

# Serve on your network
./zimserver --host 0.0.0.0 --port 8080 /path/to/zims
```

Open `http://localhost:8080` in your browser. That's it.

## ZIM files
Download from [download.kiwix.org](https://download.kiwix.org/zim/) - Wikipedia, Wiktionary, medical references, Stack Exchange, TED, books, and more.

## License

This program is free software: you can redistribute it and/or modify it under the terms of the GNU Affero General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License along with this program. If not, see http://www.gnu.org/licenses/.