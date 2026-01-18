# mc - SSH Connection Manager

`~/.ssh/config`에 등록된 호스트를 fuzzy finder로 빠르게 선택하여 SSH 접속할 수 있는 CLI 도구입니다.

## Demo

![demo](./demo.gif)

## 설치

### macOS (Homebrew)

```bash
brew tap zbum/tap
brew install zbum/tap/mc
```

### 소스에서 빌드

```bash
go install
```

또는

```bash
make build
```

## 사용법

### 기본 사용

```bash
mc
```

호스트 목록이 표시되면:
- 타이핑하여 검색
- 화살표 키로 이동
- Enter로 선택
- ESC 또는 Ctrl+C로 취소

### 초기 검색어 지정

```bash
mc prod      # "prod"로 필터링된 상태로 시작
mc web api   # "web api"로 필터링
```

## SSH Config 예시

`~/.ssh/config`:

```
# Production 서버
Host prod-web
    HostName web.prod.example.com
    User admin
    Port 22
    IdentityFile ~/.ssh/prod_key

# Development 서버
Host dev-web
    HostName web.dev.example.com
    User developer
```

## 디버그 모드

연결 문제 해결 시:

```bash
MC_DEBUG=1 mc
```

## 키보드 단축키

| 키 | 동작 |
|---|---|
| Enter | 선택한 호스트에 접속 |
| ESC / Ctrl+C | 취소 |
| 화살표 위/아래 | 호스트 이동 |
| 타이핑 | 호스트 검색 |

## 인증 순서

1. SSH config에 지정된 `IdentityFile`
2. SSH Agent
3. 기본 키 파일 (`~/.ssh/id_ed25519`, `id_rsa` 등)
4. 패스워드 입력

## 라이선스

MIT
