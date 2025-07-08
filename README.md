# TermShare

**Collaborative Terminal Sharing Tool in Go**

TermShare is a lightweight, interactive terminal sharing application written in Go. It enables a user to host a terminal session that others can join to view or collaborate in real-time. Inspired by tools like `tmate`, it supports multiple clients with one active editor at a time — ideal for remote pair programming, interviews, or command-line demos.

---

## 🚀 Features

- 🖥️ Live terminal sharing with multiple connected clients
- ✍️ Single-editor mode: only one client can send input at a time
- 🔁 Request/grant editor control functionality
- 🔌 Custom lightweight protocol over TCP
- 🛠️ PTY-backed terminal session management
- 🧱 Modular Go codebase (client/server/shared/pty)

---

## 📦 Project Structure

![image](https://github.com/user-attachments/assets/c8b2a05b-302c-4ffa-ab72-71f625f4ca43)



---

## 🧪 Getting Started

### 📋 Prerequisites

- Go 1.20+
- Linux/macOS (PTY support required)

### 🔧 Installation

```bash
git clone https://github.com/SidhantKaul/TermShare.git
cd TermShare

go build -o termshare main.go

**SAMPLE FLOWS**

Initialize TermShare as:
1. Host
2. Client
Enter your choice (1 or 2): 1
Enter the IP and port to listen on (e.g., 127.0.0.1:9000): 127.0.0.1:9000

**On a different terminal:**

Initialize TermShare as:
1. Host
2. Client
Enter your choice (1 or 2): 2
Enter the host address (e.g., 127.0.0.1:9000): 127.0.0.1:9000
Enter your name: alice
