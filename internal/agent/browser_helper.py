#!/usr/bin/env python3
import base64
import ipaddress
import json
import os
import pathlib
import re
import socket
import sys
from urllib.parse import urlparse

def fail(message):
    raise RuntimeError(message)


def validate_url(value):
    parsed = urlparse(value)
    if parsed.scheme not in {"http", "https"} or not parsed.hostname:
        fail("browser only permits http and https URLs")
    if parsed.username or parsed.password:
        fail("browser URLs must not contain embedded credentials")
    try:
        port = parsed.port or (80 if parsed.scheme == "http" else 443)
    except ValueError:
        fail("browser URL has an invalid port")
    try:
        addresses = socket.getaddrinfo(parsed.hostname, port, type=socket.SOCK_STREAM)
    except OSError:
        fail("browser DNS resolution failed")
    if not addresses:
        fail("browser DNS resolution returned no addresses")

    allow_loopback = os.environ.get("OLLAMA_AGENT_BROWSER_ALLOW_PRIVATE", "") == "1"
    for address in addresses:
        try:
            ip = ipaddress.ip_address(address[4][0].split("%", 1)[0])
        except ValueError:
            fail("browser DNS resolution returned an invalid address")
        if allow_loopback and ip.is_loopback:
            continue
        if not ip.is_global:
            fail("browser URL resolves to a non-public address")


def validate_cdp_url(value):
    # The Browser Operator may attach to a Chrome the user launched with a
    # remote-debugging port (so it can act within the user's own, logged-in
    # session). For safety this is restricted to a LOOPBACK debugger: Hades
    # never attaches to a remote browser, and each action still goes through the
    # per-step human approval gate and the same anti-SSRF route guard below.
    parsed = urlparse(value)
    if parsed.scheme not in {"http", "https"} or not parsed.hostname:
        fail("browser CDP URL must be an http(s) endpoint")
    try:
        port = parsed.port or (80 if parsed.scheme == "http" else 443)
    except ValueError:
        fail("browser CDP URL has an invalid port")
    try:
        addresses = socket.getaddrinfo(parsed.hostname, port, type=socket.SOCK_STREAM)
    except OSError:
        fail("browser CDP host did not resolve")
    if not addresses:
        fail("browser CDP host resolved to no addresses")
    for address in addresses:
        try:
            ip = ipaddress.ip_address(address[4][0].split("%", 1)[0])
        except ValueError:
            fail("browser CDP host resolved to an invalid address")
        if not ip.is_loopback:
            fail("browser CDP URL must point to a loopback debugger (127.0.0.1)")


def browser_headless():
    # Default to headless. Set OLLAMA_AGENT_BROWSER_HEADLESS=0 to launch a
    # visible window (useful when the user drives/logs in alongside the agent).
    return os.environ.get("OLLAMA_AGENT_BROWSER_HEADLESS", "1").strip() != "0"


def guard_browser_route(route):
    try:
        validate_url(route.request.url)
    except Exception:
        route.abort(error_code="blockedbyclient")
        return
    route.continue_()


def page_for(context):
    if context.pages:
        return context.pages[0]
    return context.new_page()


def save_session_state(path, page):
    if page.url.startswith(("http://", "https://")):
        path.write_text(json.dumps({"url": page.url}, ensure_ascii=False), encoding="utf-8")


def browser_executable():
    configured = os.environ.get("OLLAMA_AGENT_BROWSER_EXECUTABLE", "").strip()
    candidates = [configured, "/usr/bin/chromium", "/usr/bin/chromium-browser", "/usr/bin/google-chrome"]
    for candidate in candidates:
        if candidate and pathlib.Path(candidate).is_file():
            return candidate
    try:
        from playwright.sync_api import sync_playwright

        with sync_playwright() as playwright:
            managed = pathlib.Path(playwright.chromium.executable_path)
        if managed.is_file():
            return str(managed)
    except Exception:
        pass
    fail("no Chromium executable is available; install Playwright Chromium or set OLLAMA_AGENT_BROWSER_EXECUTABLE")


def page_result(session_id, page, state_path, **extra):
    result = {"session_id": session_id, "url": page.url, "title": page.title()}
    result.update(extra)
    if "screenshot" not in result:
        try:
            screenshot_bytes = page.screenshot(type="jpeg", quality=55)
            result["screenshot"] = "data:image/jpeg;base64," + base64.b64encode(screenshot_bytes).decode("ascii")
        except Exception:
            pass
    save_session_state(state_path, page)
    return result


def main(request):
    from playwright.sync_api import sync_playwright

    action = str(request.get("action", "")).strip()
    session_id = str(request.get("session_id", "")).strip()
    if not re.fullmatch(r"[A-Za-z0-9_-]{1,64}", session_id):
        fail("invalid browser session id")
    root = pathlib.Path(os.environ.get("OLLAMA_AGENT_BROWSER_ROOT", "/tmp/ollama-agent-browser"))
    user_dir = root / "sessions" / session_id
    user_dir.mkdir(parents=True, mode=0o700, exist_ok=True)
    state_path = user_dir / "state.json"
    cdp_url = os.environ.get("OLLAMA_AGENT_BROWSER_CDP_URL", "").strip()

    with sync_playwright() as playwright:
        own_context = True
        if cdp_url:
            # Attach to the user's running Chrome (loopback debugger) so the
            # agent operates within the user's real, logged-in session.
            validate_cdp_url(cdp_url)
            browser = playwright.chromium.connect_over_cdp(cdp_url)
            browser_context = browser.contexts[0] if browser.contexts else browser.new_context(accept_downloads=True)
            own_context = False
        else:
            executable = browser_executable()
            browser_context = playwright.chromium.launch_persistent_context(
                str(user_dir), headless=browser_headless(), executable_path=executable,
                accept_downloads=True, service_workers="block",
                args=["--disable-dev-shm-usage", "--disable-background-networking", "--disable-sync"],
            )
        try:
            browser_context.route("**/*", guard_browser_route)
            page = page_for(browser_context)
            timeout = min(max(int(request.get("timeout_ms", 15000)), 1000), 60000)
            page.set_default_timeout(timeout)
            if page.url == "about:blank" and state_path.is_file():
                previous = json.loads(state_path.read_text(encoding="utf-8")).get("url", "")
                if previous:
                    validate_url(previous)
                    page.goto(previous, wait_until="domcontentloaded")
            if action == "navigate":
                url = str(request.get("url", "")).strip()
                validate_url(url)
                page.goto(url, wait_until="domcontentloaded")
                validate_url(page.url)
                return page_result(session_id, page, state_path)
            if action in {"snapshot", "content"}:
                validate_url(page.url)
                content = page.locator("body").inner_text(timeout=timeout)
                return page_result(session_id, page, state_path, content=content[:131072])
            if action == "click":
                page.locator(str(request.get("selector", ""))).click()
                validate_url(page.url)
                return page_result(session_id, page, state_path)
            if action == "fill":
                page.locator(str(request.get("selector", ""))).fill(str(request.get("text", "")))
                return page_result(session_id, page, state_path)
            if action == "press":
                page.locator(str(request.get("selector", ""))).press(str(request.get("key", "Enter")))
                validate_url(page.url)
                return page_result(session_id, page, state_path)
            if action == "upload":
                source = pathlib.Path(str(request.get("path", ""))).resolve()
                if not source.is_file():
                    fail("upload path is not a file")
                page.locator(str(request.get("selector", ""))).set_input_files(str(source))
                return page_result(session_id, page, state_path, uploaded=source.name)
            if action == "download":
                target = pathlib.Path(str(request.get("save_path", ""))).resolve()
                target.parent.mkdir(parents=True, exist_ok=True)
                with page.expect_download(timeout=timeout) as download_info:
                    page.locator(str(request.get("selector", ""))).click()
                download = download_info.value
                download.save_as(str(target))
                return page_result(session_id, page, state_path, path=str(target), suggested_filename=download.suggested_filename)
            if action == "screenshot":
                target = pathlib.Path(str(request.get("save_path", ""))).resolve()
                target.parent.mkdir(parents=True, exist_ok=True)
                page.screenshot(path=str(target), full_page=bool(request.get("full_page", True)))
                return page_result(session_id, page, state_path, path=str(target))
            if action == "takeover":
                return {"session_id": session_id, "status": "approval_required", "reason": "human takeover must be completed by a connected Desktop/Browser surface", "url": page.url}
            fail(f"unsupported browser action: {action}")
        finally:
            # Never close the user's own browser when attached over CDP; only
            # tear down contexts we launched ourselves.
            if own_context:
                browser_context.close()


if __name__ == "__main__":
    try:
        print(json.dumps(main(json.load(sys.stdin)), ensure_ascii=False))
    except Exception as exc:
        print(json.dumps({"error": str(exc)}, ensure_ascii=False))
        sys.exit(1)
