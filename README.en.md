Cartoon Scheduler
=================

Русская версия: [README.md](README.md)

This project automates cartoon broadcasting through OBS Studio. It scans the local media library, builds daily schedules, and manages the `VideoSource` input inside OBS to play the correct episode at the right time. The toolkit includes three console utilities:

- `cartoon-scheduler` — the main web service that integrates with OBS.
- `cartoon-scanner` — helper utility from the `tools/` folder that produces `cartoons.json`.
- `parse_schedule` — interactive CLI from `generate/` that builds `schedule_*.json`.

---

Requirements
------------

- Go **1.22+** (the `go.mod` declares `go 1.25.3`; use the latest stable release such as 1.22 or newer).
- OBS Studio 29+ with obs-websocket enabled on port **4455** and password `123456`, or adjust `obsPassword` in `main.go`.
- `ffprobe` (part of FFmpeg) available in `PATH` or placed next to the executables.
- Local `cartoons/` directory with subfolders per cartoon (`cartoons/<cartoon_id>/<video-file>`).

---

Project layout
--------------

- `main.go` — web server with `/start`, `/scan`, `/stop` handlers that control OBS.
- `tools/generate_meta.go` — source code of the `cartoon-scanner` utility, extracts metadata and generates `cartoons.json`.
- `generate/main.go` — source code of `parse_schedule`, creates `schedule_morning.json`, `schedule_day.json`, `schedule_evening.json`.
- `cartoons/` — media library of cartoons.
- `cartoons.json`, `schedule_*.json`, `state.json` — shared data artifacts.
- `go.mod`, `go.sum` — Go module definition and dependencies (`goobs` for OBS control, `mpb` for progress bars).

---

Installing Go and dependencies
------------------------------

1. **Install Go**
   - macOS: `brew install go`
   - Windows: download the MSI installer from https://go.dev/dl/

2. **Verify your environment**
   - Run `go version` to confirm Go is available.
   - Update `PATH` / `GOPATH` if necessary.

3. **Download project dependencies**
   ```sh
   cd /Users/andrei/Documents/develop
   go mod tidy
   ```
   This installs all modules declared in `go.mod` and refreshes `go.sum`.

---

Workspace rules
---------------

For consistent behavior, keep **all executables and data files in the same directory** (for example `C:\cartoon-scheduler\` or `/Users/andrei/Documents/develop`). Minimum set:

- `cartoon-scheduler.exe` (Windows) or `cartoon-scheduler` (macOS/Linux)
- `cartoon-scanner.exe` / `cartoon-scanner`
- `schedule_parse.exe` / `parse_schedule`
- `ffprobe.exe` (Windows) or `ffprobe` accessible via `PATH`
- `cartoons/` (structured as `cartoons/<cartoon_id>/<files>`)
- `cartoons.json` (produced by `cartoon-scanner`)
- `schedule_morning.json`, `schedule_day.json`, `schedule_evening.json` (produced by `parse_schedule`)
- `state.json` (created automatically by the main service)

---

Building binaries
-----------------

Navigate to the project root first:
```sh
cd /Users/andrei/Documents/develop
```

### 1. Main service (`cartoon-scheduler`)
- **macOS / Linux**
  ```sh
  go build -o cartoon-scheduler .
  ```
- **Windows**
  ```sh
  GOOS=windows GOARCH=amd64 go build -o cartoon-scheduler.exe .
  ```

### 2. Library scanner (`cartoon-scanner`)
- **macOS / Linux**
  ```sh
  go build -o cartoon-scanner ./tools
  ```
- **Windows**
  ```sh
  GOOS=windows GOARCH=amd64 go build -o cartoon-scanner.exe ./tools
  ```

### 3. Schedule generator (`parse_schedule`)
- **macOS / Linux**
  ```sh
  go build -o parse_schedule ./generate
  ```
- **Windows**
  ```sh
  GOOS=windows GOARCH=amd64 go build -o schedule_parse.exe ./generate
  ```

> Tip: Run the `GOOS=windows` commands on macOS to cross-compile `.exe` binaries without a virtual machine.

---

Utility documentation
---------------------

### cartoon-scanner (`tools/generate_meta.go`)
1. Run `cartoon-scanner` from the directory that contains the `cartoons/` folder with its subdirectories.
2. The utility scans each subfolder (`cartoons/<cartoon_id>/…`), measures video duration via `ffprobe`, and collects episode metadata.
3. When finished, it writes `cartoons.json` next to the executable and prints a summary of the discovered cartoons.
4. If `cartoons/` is missing or empty, the tool exits with a hint to create the folder structure and add files.

### parse_schedule (`generate/main.go`)
1. Make sure an up-to-date `cartoons.json` is available next to the executable.
2. Launch `parse_schedule` (on Windows: `schedule_parse.exe`). The CLI greets you with the list of available `cartoon_id` values.
3. Enter pairs `cartoon_id number_of_episodes`. Type `exit` to stop.
4. The script fills three periods in order — morning (06:00–12:00), day (12:00–18:00), evening (18:00–00:00). Keep entering cartoons and counts until the whole day is filled; the program notifies you when there is no room left.
5. The resulting `schedule_morning.json`, `schedule_day.json`, and `schedule_evening.json` are saved alongside the binary.

### cartoon-scheduler (`main.go`)
1. Prepare OBS Studio:
   - enable the obs-websocket server (default port 4455);
   - set the password to `123456` or adjust `obsPassword` in the source code;
   - create a media source named `VideoSource` in the target scene.
2. Ensure the working directory contains:
   - the `cartoon-scheduler` binary (`.exe` on Windows);
   - `cartoon-scanner` (triggered by the “Scan cartoons” button);
   - `cartoons.json`, `schedule_*.json`, `state.json` (the last file appears after the first run);
   - the `cartoons/` folder with media files.
3. Start `cartoon-scheduler`. The console prints `http://localhost:8080`.
4. Open the web interface: “Start schedule” kicks off playback based on the current period; “Scan cartoons” invokes `cartoon-scanner`; “Stop” terminates the loop.
5. The app updates `state.json` after each episode so it can resume from the right point next time.

---

Operational workflow
--------------------

1. **Prepare the media library**  
   Store videos inside `cartoons/<cartoon_id>/episode.ext`. The folder name becomes the `cartoon_id` used later in schedules.

2. **Generate the cartoon catalog**  
   Run `./cartoon-scanner` to produce `cartoons.json`.

3. **Create the daily schedules**  
   Run `./parse_schedule` and keep entering `cartoon_id count` pairs until the entire day is filled.

4. **Start the broadcast**  
   Run `./cartoon-scheduler` and control playback via http://localhost:8080.

---

Examples
--------

- **Directory layout of `cartoons/`**

```text
cartoons/
├── batmen/
│   ├── The.Batman.S01E01.mkv
│   ├── The.Batman.S01E02.mkv
│   └── ...
├── futurama/
│   ├── 103 - I, Roommate.mkv
│   ├── 104 - Love's Labours Lost in Space.mkv
│   └── ...
└── ...
```

- **Sample `cartoons.json` generated by the scanner**

```json
{
  "batmen": {
    "title": "Batmen",
    "total_episodes": 6,
    "episodes": [
      {
        "filename": "The.Batman.S01E01.mkv",
        "title": "The Batman — S01E01",
        "duration": 1320
      },
      {
        "filename": "The.Batman.S01E02.mkv",
        "title": "The Batman — S01E02",
        "duration": 1344
      }
    ]
  },
  "futurama": {
    "title": "Futurama",
    "total_episodes": 10,
    "episodes": [
      {
        "filename": "103 - I, Roommate.mkv",
        "title": "103 - I, Roommate",
        "duration": 1498
      }
    ]
  }
}
```

- **Sample `schedule_morning.json` produced by `parse_schedule`**

```json
{
  "period_name": "morning",
  "start_time": "06:00",
  "end_time": "12:00",
  "slots": [
    {
      "cartoon_id": "batmen",
      "start": "06:00",
      "end": "07:05"
    },
    {
      "cartoon_id": "futurama",
      "start": "07:05",
      "end": "08:35"
    }
  ]
}
```

---

Useful commands
---------------

- Clear the module cache: `go clean -modcache`
- List dependencies: `go list -m all`
- Upgrade obs client library: `go get github.com/andreykaipov/goobs@latest`

---

Troubleshooting
---------------

If you meet build or runtime issues (e.g. “ffprobe not found”, OBS connection failures), inspect the console messages — each program prints actionable hints on what to fix.

---

Developer contact
-----------------

- Telegram: [@danilllenko](https://t.me/danilllenko)
- Email: [danilko.a.g@gmail.com](mailto:danilko.a.g@gmail.com)

