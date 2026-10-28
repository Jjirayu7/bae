# Bae

Custom agent harness integrate with agy provider

---

## Quickstart

```bash
# 1. Clone repository
git clone <your-repo-url>
cd bae

# 2. Install (คอมไพล์และติดตั้งลง ~/.local/bin ทันที)
./install.sh
# หรือใช้ make install

# 3. ตรวจสอบความพร้อมของระบบ
make doctor

# 4. ใช้งาน
bae
```

---

## Commands

| คำสั่ง | รายละเอียด |
| :--- | :--- |
| **`/skill [name]`** | ค้นหาและโหลด Skill (เช่น `/skill tdd`, `/skill code-review`) จาก `~/.agents/skills` เข้ามาใน Context |
| **`/learn <rule>`** | บันทึกกฎ/สไตล์การโค้ดส่วนตัวลง Memory ถาวร (`~/.gemini/bae/preferences.md` / OpenViking) |
| **`/remember [q]`** | ค้นหาหรือดึงความจำ/ข้อตกลงที่เคยสอนไว้ |
| **`/rewind`** | ย้อนกลับ 1 เทิร์น (ลบข้อความ Prompt & Response ล่าสุดออกจากประวัติ) |
| **`/compact`** | บีบอัดประวัติแชท เคลียร์ข้อความเก่าเพื่อประหยัด Token Window |
| **`/diff`** | ดู Git Diff สี (Unstaged code changes) แบบ Real-time |
| **`/status`** | ดูสถานะ Git (`git status --short`) |
| **`/doctor`** | วินิจฉัยความพร้อมของ Ecosystem (Go, RTK, OV, CCC, Graphify, AGY) |
| **`/tools`** | แสดงรายการเครื่องมือทั้งหมดของ Agent |
| **`/clear`** | ล้างหน้าจอแชท |
| **`/exit`** | ออกจากโปรแกรม (หรือกด `Ctrl + C`) |

---

## Project Structure

```
bae/
├── Makefile                   # build, install, doctor, clean
├── install.sh                 # One-click installer
├── cmd/
│   └── bae/
│       └── main.go            # Entry point, Flags, TUI Launcher
├── internal/
│   ├── agent/
│   │   └── agent.go           # Dual Provider Engine + Live Streaming + Caveman
│   ├── tools/
│   │   └── tools.go           # Native tools (RTK, OV, CCC, Graphify, Edits)
│   └── ui/
│       ├── chat.go            # Single-row Input Bubble Tea TUI
│       ├── commands.go        # Slash Commands (/skill, /learn, /rewind, etc.)
│       ├── diff.go            # Unified Code Diff Formatter
│       └── statusline.go      # ASCII Statusline
├── go.mod
├── go.sum
└── README.md
```
