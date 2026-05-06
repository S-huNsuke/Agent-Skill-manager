"""Playwright test script for Agent Skills Manager frontend."""
import json
import sys
from playwright.sync_api import sync_playwright

BASE_URL = "http://localhost:5173"
ROUTES = ["/", "/agents", "/skills", "/store", "/projects", "/assistant", "/settings"]
ROUTE_NAMES = ["首页", "代理", "技能", "商店", "项目", "助手", "设置"]

results = {
    "pages": {},
    "navigation": {},
    "console_errors": [],
    "summary": {}
}


def test_app():
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        context = browser.new_context(viewport={"width": 1440, "height": 960})
        page = context.new_page()

        page.on("console", lambda msg: _on_console(msg))

        print("=" * 60)
        print("Agent Skills Manager - 功能测试")
        print("=" * 60)

        print("\n[1] 测试首页加载...")
        page.goto(f"{BASE_URL}/")
        page.wait_for_load_state("networkidle")
        page.wait_for_timeout(1000)

        _screenshot(page, "test-home.png")
        _test_home_page(page)

        print("\n[2] 测试侧边栏导航...")
        _test_sidebar_navigation(page)

        print("\n[3] 逐页测试各路由...")
        for route, name in zip(ROUTES, ROUTE_NAMES):
            _test_route(page, route, name)

        print("\n[4] 检查控制台错误...")
        _check_console_errors()

        print("\n[5] 生成测试报告...")
        _generate_report()

        browser.close()


def _on_console(msg):
    if msg.type in ("error", "warning"):
        entry = {"type": msg.type, "text": msg.text}
        results["console_errors"].append(entry)


def _screenshot(page, path):
    try:
        out = f"test-screenshots/{path}"
        page.screenshot(path=out, full_page=True)
        print(f"  截图保存: {out}")
    except Exception as e:
        print(f"  截图失败: {e}")


def _test_home_page(page):
    title = page.title()
    print(f"  页面标题: {title}")

    body_text = page.locator("body").inner_text()
    has_content = len(body_text.strip()) > 50
    print(f"  页面内容: {'有内容' if has_content else '内容不足'} ({len(body_text)} 字符)")

    results["pages"]["/"] = {
        "title": title,
        "has_content": has_content,
        "content_length": len(body_text)
    }


def _test_sidebar_navigation(page):
    page.goto(f"{BASE_URL}/")
    page.wait_for_load_state("networkidle")
    page.wait_for_timeout(500)

    nav_links = page.locator("nav a, aside a, [class*='sidebar'] a, [class*='nav'] a").all()
    print(f"  找到 {len(nav_links)} 个导航链接")

    for i, link in enumerate(nav_links):
        try:
            text = link.inner_text().strip()
            href = link.get_attribute("href") or ""
            print(f"  链接 {i + 1}: '{text}' -> {href}")
            results["navigation"][f"link_{i}"] = {"text": text, "href": href}
        except Exception as e:
            print(f"  链接 {i + 1}: 读取失败 - {e}")


def _test_route(page, route, name):
    print(f"\n  测试路由: {route} ({name})")
    try:
        url = f"{BASE_URL}#{route}" if route != "/" else f"{BASE_URL}/"
        page.goto(url)
        page.wait_for_load_state("networkidle")
        page.wait_for_timeout(1500)

        _screenshot(page, f"route-{route.strip('/') or 'home'}.png")

        body_text = page.locator("body").inner_text()
        has_content = len(body_text.strip()) > 50

        heading_count = page.locator("h1, h2, h3").count()

        page_title = page.title()

        status = "通过" if has_content else "内容不足"
        print(f"  状态: {status}")
        print(f"  标题: {page_title}")
        print(f"  内容长度: {len(body_text)} 字符")
        print(f"  标题元素: {heading_count} 个")

        results["pages"][route] = {
            "name": name,
            "status": status,
            "has_content": has_content,
            "content_length": len(body_text),
            "heading_count": heading_count,
            "page_title": page_title
        }
    except Exception as e:
        print(f"  错误: {e}")
        results["pages"][route] = {"name": name, "status": "失败", "error": str(e)}


def _check_console_errors():
    errors = results["console_errors"]
    error_count = len([e for e in errors if e["type"] == "error"])
    warning_count = len([e for e in errors if e["type"] == "warning"])

    print(f"  控制台错误: {error_count} 个")
    print(f"  控制台警告: {warning_count} 个")

    for e in errors[:10]:
        prefix = "ERROR" if e["type"] == "error" else "WARN"
        text = e["text"][:120]
        print(f"  [{prefix}] {text}")


def _generate_report():
    pages = results["pages"]
    total = len(pages)
    passed = sum(1 for p in pages.values() if p.get("status") == "通过")
    failed = total - passed

    results["summary"] = {
        "total_pages": total,
        "passed": passed,
        "failed": failed,
        "console_errors": len([e for e in results["console_errors"] if e["type"] == "error"]),
        "console_warnings": len([e for e in results["console_errors"] if e["type"] == "warning"])
    }

    print("\n" + "=" * 60)
    print("测试报告")
    print("=" * 60)
    print(f"  总页面数: {total}")
    print(f"  通过: {passed}")
    print(f"  失败: {failed}")
    print(f"  控制台错误: {results['summary']['console_errors']}")
    print(f"  控制台警告: {results['summary']['console_warnings']}")

    print("\n  各页面详情:")
    for route, info in pages.items():
        name = info.get("name", route)
        status = info.get("status", "未知")
        content_len = info.get("content_length", 0)
        icon = "✅" if status == "通过" else "❌"
        print(f"    {icon} {name} ({route}): {status}, {content_len} 字符")


if __name__ == "__main__":
    test_app()
