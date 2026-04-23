<p align="center">
  <img src="agent-message-logo.svg" alt="Agent Message logo" width="88">
</p>

# Agent Message

[English](README.md) | [简体中文](README.zh-CN.md) | [繁體中文](README.zh-TW.md) | [日本語](README.ja.md)

<p align="center">
  <img src="docs/readme-screenshot.png" alt="Agent Message screenshot" width="900">
</p>

Agent Message는 에이전트와 사람이 같은 DM 흐름에서 작업하기 위한 메시징 스택입니다.

- HTTP/SSE 서버 (`server/`)
- 웹 앱 (`web/`)
- CLI (`cli/`)

## 왜 Agent Message인가

- CLI로 노출되어 에이전트가 직접 메시지를 보내고 읽고 감시할 수 있습니다.
- `json_render`를 통해 카드, 표, 배지, 진행 상태 같은 읽기 좋은 구조화 메시지를 받을 수 있습니다.
- 웹 앱을 휴대폰에서 열어 에이전트와 작업을 이어갈 수 있습니다.
- `codex-message`와 `claude-message`는 Codex 및 Claude 세션을 같은 DM 워크플로로 연결합니다.

## 래퍼 패키지

- `codex-message`: Codex app-server 세션을 시작하고 Agent Message DM으로 대화를 중계합니다.
- `claude-message`: Claude CLI 세션을 시작하고 프롬프트, 실패, 최종 답변을 Agent Message로 중계합니다.

랜딩 페이지는 `https://amessage.dev`에서 볼 수 있습니다. 호스팅 클라우드 서비스는 아직 준비 중입니다. 현재 권장 방식은 self-hosted local stack입니다.

## 지원 플랫폼

현재 npm 패키지는 macOS 빌드만 배포합니다.

| 플랫폼 | 아키텍처 | `agent-message` | `codex-message` | `claude-message` | 비고 |
| --- | --- | --- | --- | --- | --- |
| macOS | Apple Silicon (`arm64`) | 지원 | 지원 | 지원 | 기본 패키징 대상 |
| macOS | Intel (`x64`) | 지원 | 지원 | 지원 | 패키징 대상 |
| Linux | `x64` / `arm64` | 패키지 없음 | 패키지 없음 | 패키지 없음 | 소스 빌드만 가능 |
| Windows | `x64` / `arm64` | 패키지 없음 | 패키지 없음 | 패키지 없음 | 현재 미지원 |

## Setup Prompt

Claude Code 또는 Codex에 그대로 붙여 넣으세요.

```bash
Set up https://github.com/siisee11/agent-message for me.

Read `install.md` and follow the self-host setup flow. Ask me for the account-id before registering, use 0000 only as the temporary initial password, remind me to change it immediately, set the master recipient, and send me a welcome message with agent-message when setup is complete.
```

## 빠른 설정

클라우드 서비스 계정은 아직 제공되지 않습니다. 지금은 self-hosted local stack을 사용하세요.

에이전트가 Agent Message를 대신 설치한다면 위의 Setup Prompt를 사용하세요.

직접 설치하려면 npm을 사용할 수 있습니다.

```bash
npx skills add https://github.com/siisee11/agent-message --skill agent-message-cli -g -y
npm install -g agent-message
agent-message start
agent-message status
```

그다음 로컬 계정을 만들거나 로그인합니다. 등록하기 전에 사용자에게 `account-id`를 먼저 물어보세요. `0000`은 임시 초기 비밀번호로만 사용해야 합니다.

```bash
agent-message register <account-id> 0000
# 계정이 이미 있다면:
agent-message login <account-id> 0000
```

브라우저에서 `http://127.0.0.1:45788`을 열고 Profile 페이지에서 비밀번호를 즉시 변경하세요.
`agent-message start`는 로컬 스택을 실행하고 CLI 트래픽이 시작된 API (`http://127.0.0.1:45180`)로 향하도록 `~/.agent-message/config`를 갱신합니다.

공개 표시 이름과 에이전트 상태 보고를 받을 기본 수신자를 설정합니다.

```bash
agent-message username set <username>
agent-message config set master <recipient-username>
agent-message whoami
```

## npm 설치 (macOS)

macOS (`arm64`, `x64`)에서는 npm으로 패키징된 앱을 설치할 수 있습니다.

```bash
npm install -g agent-message
agent-message start
agent-message status
agent-message stop
agent-message upgrade
agent-message uninstall
```

기본 포트:
- API: `127.0.0.1:45180`
- Web: `127.0.0.1:45788`

self-hosted local 사용 시 `agent-message start`는 기본적으로 로컬 SQLite 데이터베이스를 생성해 사용합니다. 클라우드 서비스는 아직 준비 중입니다. 향후 managed cloud 배포는 `DB_DRIVER=postgres`와 `POSTGRES_DSN`으로 서버를 실행해야 합니다.
`agent-message uninstall`은 로컬 스택을 멈추고 글로벌 npm 패키지를 제거합니다. 기본적으로 `~/.agent-message`는 보존하므로 로컬 계정, SQLite 데이터, 업로드, 로그, CLI 프로필이 실수로 삭제되지 않습니다. 런타임 데이터까지 삭제하려면 `agent-message uninstall --purge`를 실행하세요.

일반 CLI 명령은 같은 `agent-message` 명령에서 계속 사용할 수 있습니다.

```bash
agent-message register alice 0000
agent-message login alice 0000
agent-message username set jay-ui-bot
agent-message config set master jay
agent-message ls
agent-message open bob
agent-message send bob "hello"
agent-message send "status update for master"
```

필요하면 런타임 위치와 포트를 바꿀 수 있습니다.

```bash
agent-message start --runtime-dir /tmp/agent-message --api-port 28080 --web-port 28788
agent-message status --runtime-dir /tmp/agent-message --api-port 28080 --web-port 28788
agent-message stop --runtime-dir /tmp/agent-message
```

## 웹 개발

```bash
cd web
npm ci
npm run dev
```

로컬 개발에서 Vite는 `/api/...`와 `/static/uploads/...`를 `http://localhost:8080`으로 프록시합니다. API가 다른 origin에 있다면 `VITE_API_BASE_URL`을 설정하세요.

```bash
cd web
VITE_API_BASE_URL=http://localhost:8080 npm run dev
```

빌드 확인:

```bash
cd web
npm run build
```

## Cloudflare Workers 웹 배포

현재 React 웹 앱은 Cloudflare Workers의 static assets로 배포할 수 있습니다. 우선 공개 웹 표면을 올리는 용도이며, API 기반 클라우드 서비스는 나중에 연결할 수 있습니다.

```bash
cd web
npm ci
npm run deploy:worker
```

클라우드 API가 준비되기 전까지 Worker의 API 요청은 `503`을 반환합니다. API origin이 생기면 Worker에 `API_ORIGIN`을 설정하고 다시 배포하세요.

```bash
cd web
npx wrangler secret put API_ORIGIN
npm run deploy:worker
```

예를 들어 `API_ORIGIN=https://api.amessage.dev`를 설정하면 웹 앱은 같은 origin의 `/api/...` 호출을 유지하고 Worker가 API 서비스로 프록시합니다.

## 소스에서 실행

체크아웃한 저장소를 `agent-message` 명령으로 사용하려면 저장소 루트에서 다음을 실행합니다.

```bash
npm link
```

로컬 소스 트리로 실행하려면 `--dev`를 추가합니다.

```bash
agent-message start --dev
agent-message stop --dev
```

필요 조건:
- Go `1.26+`
- Node.js `18+` 및 npm
- Docker + Docker Compose (self-host container 흐름용)

## 서버 빠른 시작

SQLite 로컬 서버:

```bash
cd server
go run .
```

## Self-host Container Deploy

홈 Mac 서버에서는 컨테이너만으로 self-hosted stack을 실행할 수 있습니다.

```bash
cp .env.selfhost.example .env.selfhost
make publish
docker compose --env-file .env.selfhost -f docker-compose.selfhost.yml ps
docker compose --env-file .env.selfhost -f docker-compose.selfhost.yml logs -f
make unpublish
```

필수 값:
- `APP_HOSTNAME`
- `POSTGRES_PASSWORD`
- `CLOUDFLARE_TUNNEL_TOKEN`

스택 구성:
- `postgres`
- `server`
- `gateway`
- `cloudflared`

Mac에서 host port를 공개할 필요는 없습니다. 공개 트래픽은 Cloudflare Tunnel을 통해 들어와야 합니다.

## Claude Code Skill

Agent Message CLI skill을 설치하면 Claude Code가 이 프로젝트의 CLI 명령, 플래그, `json_render` 컴포넌트 카탈로그를 이해할 수 있습니다.

```bash
npx skills add https://github.com/siisee11/agent-message --skill agent-message-cli
```

## codex-message

`codex-message`는 Codex 예제 앱입니다. `codex app-server`를 감싸고 `agent-message`를 DM transport로 사용합니다.

```bash
npm install -g agent-message codex-message
```

필요 조건:
- `agent-message` 설치 및 로그인 완료
- 대상 사용자가 이미 `agent-message` 계정을 가지고 있음
- `codex` CLI 설치 및 인증 완료

일반 설정:

```bash
agent-message config set master jay
codex-message --model gpt-5.4 --cwd /path/to/worktree
codex-message --model gpt-5.4 --cwd /path/to/worktree --yolo
codex-message --to alice --model gpt-5.4 --cwd /path/to/worktree
codex-message --bg --model gpt-5.4 --cwd /path/to/worktree
```

동작 방식:
- 세션마다 새로운 `agent-{chatId}` 계정을 만듭니다.
- 대상 사용자에게 생성된 자격 증명이 포함된 시작 DM을 보냅니다.
- 하나의 Codex app-server thread를 그 DM 대화에 붙입니다.
- 일반 텍스트 DM을 Codex로 전달합니다.
- 승인, 입력, 실패, 상태 프롬프트는 wrapper가 `json_render`로 보냅니다.
- 최종 사용자용 결과는 Codex가 `agent-message send --from agent-{chatId}`로 직접 보내도록 지시됩니다.

유용한 플래그:
- `--to <username>`
- `--cwd /path/to/worktree`
- `--model gpt-5.4`
- `--approval-policy on-request`
- `--sandbox workspace-write`
- `--network-access`
- `--yolo`
- `--bg`

## claude-message

`claude-message`는 Claude 예제 앱입니다. `claude -p --output-format json`을 실행하고 세션을 `agent-message`로 중계합니다.

```bash
npm install -g agent-message claude-message
```

필요 조건:
- `agent-message` 설치 및 로그인 완료
- 대상 사용자가 이미 `agent-message` 계정을 가지고 있음
- `claude` CLI 설치 및 인증 완료

예시:

```bash
claude-message --to jay --model sonnet --permission-mode accept-edits
claude-message --bg --to jay --model sonnet --permission-mode accept-edits
```

동작 방식은 `codex-message`와 비슷합니다. wrapper는 임시 `agent-{chatId}` 계정을 만들고 같은 DM 대화에서 일반 텍스트 메시지를 기다립니다. 성공한 turn에서는 에이전트가 최종 사용자용 결과를 직접 보내고, wrapper는 시작 메시지, reaction, 실패 알림을 담당합니다.

유용한 플래그:
- `--to jay`
- `--cwd /path/to/worktree`
- `--model sonnet`
- `--permission-mode accept-edits`
- `--allowed-tools Read,Edit`
- `--bare`
- `--bg`

## CLI 빠른 시작

self-hosting에서는 먼저 `agent-message start`로 로컬 스택을 시작하거나, source CLI에 `--server-url` 또는 `config set server_url`로 로컬 API를 지정하세요.

```bash
cd cli
go run . --server-url http://localhost:8080 register alice 0000
go run . --server-url http://localhost:8080 login alice 0000
go run . username set jay-ui-bot
go run . profile list
go run . profile switch alice
```

일반 명령:

```bash
go run . ls
go run . open bob
go run . send bob "hello"
go run . send bob --attach ./screenshot.png
go run . read bob --n 20
go run . edit 1 "edited text"
go run . delete 1
go run . react <message-id> 👍
go run . unreact <message-id> 👍
go run . watch bob
```

CLI config는 기본적으로 `~/.agent-message/config`에 저장됩니다. `onboard`는 사람용 interactive 명령입니다. 에이전트 설치는 `install.md`를 따라 skill 설치, npm 패키지 설치, 로컬 스택 시작, `account-id` 확인, `register` 또는 `login` 순서로 진행해야 합니다.

## 검증 및 제한

- Account ID 및 username: `3-32`자, 허용 문자 `[A-Za-z0-9._-]`
- Password: `4-72`자
- 업로드 최대 크기: `20 MB`
- 지원하지 않는 파일 형식은 거부됩니다.

## 개발 확인

```bash
cd server
go test ./...

cd cli
go test ./...

cd web
npm run build
```
