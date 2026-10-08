import importlib.util
import ipaddress
import sys
import types
import unittest
from pathlib import Path
from unittest.mock import patch


# The URL guard is intentionally testable with Python's standard library even
# when Playwright/Chromium are not installed on the developer machine.
playwright_module = types.ModuleType("playwright")
sync_api_module = types.ModuleType("playwright.sync_api")
sync_api_module.sync_playwright = lambda: None
playwright_module.sync_api = sync_api_module
sys.modules.setdefault("playwright", playwright_module)
sys.modules.setdefault("playwright.sync_api", sync_api_module)

spec = importlib.util.spec_from_file_location(
    "browser_helper", Path(__file__).with_name("browser_helper.py")
)
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)


class FakeRequest:
    def __init__(self, url):
        self.url = url


class FakeRoute:
    def __init__(self, url):
        self.request = FakeRequest(url)
        self.continued = False
        self.aborted = False

    def continue_(self):
        self.continued = True

    def abort(self, error_code=None):
        self.aborted = True
        self.error_code = error_code


class BrowserHelperSecurityTests(unittest.TestCase):
    def setUp(self):
        self.private_bypass = patch.dict("os.environ", {"OLLAMA_AGENT_BROWSER_ALLOW_PRIVATE": ""})
        self.private_bypass.start()
        self.addCleanup(self.private_bypass.stop)

    @staticmethod
    def resolved(address):
        return [(None, None, None, None, (address, 443))]

    @patch.object(helper.socket, "getaddrinfo")
    def test_public_https_url_is_allowed(self, getaddrinfo):
        getaddrinfo.return_value = self.resolved("93.184.216.34")
        helper.validate_url("https://example.com/path")
        getaddrinfo.assert_called_once_with("example.com", 443, type=helper.socket.SOCK_STREAM)

    @patch.object(helper.socket, "getaddrinfo")
    def test_private_destination_is_blocked_even_with_loopback_test_bypass(self, getaddrinfo):
        getaddrinfo.return_value = self.resolved("192.168.1.20")
        with patch.dict("os.environ", {"OLLAMA_AGENT_BROWSER_ALLOW_PRIVATE": "1"}):
            with self.assertRaises(RuntimeError):
                helper.validate_url("http://router.local/admin")

    @patch.object(helper.socket, "getaddrinfo")
    def test_loopback_test_bypass_does_not_allow_private_addresses(self, getaddrinfo):
        getaddrinfo.return_value = self.resolved("127.0.0.1")
        with patch.dict("os.environ", {"OLLAMA_AGENT_BROWSER_ALLOW_PRIVATE": "1"}):
            helper.validate_url("http://127.0.0.1:8080/")

    @patch.object(helper, "validate_url")
    def test_request_interceptor_aborts_untrusted_redirect_or_subresource(self, validate_url):
        validate_url.side_effect = RuntimeError("private destination")
        route = FakeRoute("http://127.0.0.1/private")
        helper.guard_browser_route(route)
        self.assertTrue(route.aborted)
        self.assertFalse(route.continued)

    @patch.object(helper, "validate_url")
    def test_request_interceptor_continues_validated_public_request(self, validate_url):
        route = FakeRoute("https://example.com/script.js")
        helper.guard_browser_route(route)
        validate_url.assert_called_once_with(route.request.url)
        self.assertTrue(route.continued)
        self.assertFalse(route.aborted)

    @patch.object(helper.socket, "getaddrinfo")
    def test_cdp_url_must_point_to_loopback_debugger(self, getaddrinfo):
        getaddrinfo.return_value = self.resolved("127.0.0.1")
        helper.validate_cdp_url("http://127.0.0.1:9222")

    @patch.object(helper.socket, "getaddrinfo")
    def test_cdp_url_rejects_non_loopback_debugger(self, getaddrinfo):
        getaddrinfo.return_value = self.resolved("93.184.216.34")
        with self.assertRaises(RuntimeError):
            helper.validate_cdp_url("http://evil.example.com:9222")

    def test_cdp_url_rejects_non_http_scheme(self):
        with self.assertRaises(RuntimeError):
            helper.validate_cdp_url("ftp://127.0.0.1:9222")

    def test_headless_defaults_true_and_opt_out(self):
        with patch.dict("os.environ", {"OLLAMA_AGENT_BROWSER_HEADLESS": ""}, clear=False):
            self.assertTrue(helper.browser_headless())
        with patch.dict("os.environ", {"OLLAMA_AGENT_BROWSER_HEADLESS": "0"}):
            self.assertFalse(helper.browser_headless())


if __name__ == "__main__":
    unittest.main()
