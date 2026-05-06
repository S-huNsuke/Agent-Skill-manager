"""Detailed functional test for Agent Skills Manager frontend."""
import json
from playwright.sync_api import sync_playwright

BASE_URL = "http://localhost:5173"
detailed_results = {}


def test_detailed():
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        context = browser.new_context(viewport={"width": 1440, "height": 960})
        page = context.new_page()

        console_errors = []
        page.on("console", lambda msg: console_errors.append(msg) if msg.type == "error" else None)

        print("=" * 60)
        print("Agent Skills Manager - 详细功能测试")
        print("=" * 60)

        print("\n[1] 首页 - Dashboard 功能...")
        page.goto(f"{BASE_URL}/")
        page.wait_for_load_state("networkidle")
        page.wait_for_timeout(1000)

        home_info = _analyze_page(page, "首页")
        home_info["sidebar_visible"] = page.locator("nav, aside, [class*='sidebar']").count() > 0
        home_info["sidebar_links"] = page.locator("nav a, aside a").count()

        highlights = page.locator("[class*='highlight'], [class*='card'], [class*='band']").all()
        home_info["highlight_elements"] = len(highlights)

        task_items = page.locator("[class*='task'], [class*='todo']").all()
        home_info["task_elements"] = len(task_items)

        _print_page_info("首页", home_info)
        detailed_results["首页"] = home_info

        print("\n[2] 代理页 - Agent Cards...")
        page.goto(f"{BASE_URL}#/agents")
        page.wait_for_load_state("networkidle")
        page.wait_for_timeout(1000)

        agents_info = _analyze_page(page, "代理")
        agent_cards = page.locator("[class*='agent'], [class*='card']").all()
        agents_info["agent_card_count"] = len(agent_cards)

        status_badges = page.locator("[class*='badge'], [class*='status'], [class*='tag']").all()
        agents_info["status_badge_count"] = len(status_badges)

        _print_page_info("代理", agents_info)
        detailed_results["代理"] = agents_info

        print("\n[3] 技能页 - Skill List...")
        page.goto(f"{BASE_URL}#/skills")
        page.wait_for_load_state("networkidle")
        page.wait_for_timeout(1000)

        skills_info = _analyze_page(page, "技能")
        skill_items = page.locator("[class*='skill'], [class*='card']").all()
        skills_info["skill_item_count"] = len(skill_items)

        _print_page_info("技能", skills_info)
        detailed_results["技能"] = skills_info

        print("\n[4] 商店页 - Store Items...")
        page.goto(f"{BASE_URL}#/store")
        page.wait_for_load_state("networkidle")
        page.wait_for_timeout(1000)

        store_info = _analyze_page(page, "商店")
        store_items = page.locator("[class*='store'], [class*='card']").all()
        store_info["store_item_count"] = len(store_items)

        compat_labels = page.locator("[class*='compat'], [class*='chip']").all()
        store_info["compat_label_count"] = len(compat_labels)

        _print_page_info("商店", store_info)
        detailed_results["商店"] = store_info

        print("\n[5] 项目页 - Projects...")
        page.goto(f"{BASE_URL}#/projects")
        page.wait_for_load_state("networkidle")
        page.wait_for_timeout(1000)

        projects_info = _analyze_page(page, "项目")
        project_items = page.locator("[class*='project'], [class*='card']").all()
        projects_info["project_item_count"] = len(project_items)

        _print_page_info("项目", projects_info)
        detailed_results["项目"] = projects_info

        print("\n[6] 助手页 - AI Assistant...")
        page.goto(f"{BASE_URL}#/assistant")
        page.wait_for_load_state("networkidle")
        page.wait_for_timeout(1000)

        assistant_info = _analyze_page(page, "助手")
        step_indicators = page.locator("[class*='step'], [class*='progress']").all()
        assistant_info["step_indicator_count"] = len(step_indicators)

        action_buttons = page.locator("button").all()
        assistant_info["button_count"] = len(action_buttons)

        _print_page_info("助手", assistant_info)
        detailed_results["助手"] = assistant_info

        print("\n[7] 设置页 - Diagnostics...")
        page.goto(f"{BASE_URL}#/settings")
        page.wait_for_load_state("networkidle")
        page.wait_for_timeout(1000)

        settings_info = _analyze_page(page, "设置")
        diag_items = page.locator("[class*='diagnostic'], [class*='info'], [class*='detail']").all()
        settings_info["diagnostic_item_count"] = len(diag_items)

        _print_page_info("设置", settings_info)
        detailed_results["设置"] = settings_info

        print("\n[8] 导航交互测试...")
        nav_results = _test_navigation_clicks(page)
        detailed_results["导航交互"] = nav_results

        print("\n[9] 控制台错误检查...")
        error_count = len(console_errors)
        print(f"  控制台错误总数: {error_count}")
        for err in console_errors[:5]:
            print(f"  [ERROR] {err.text[:150]}")
        detailed_results["控制台错误"] = error_count

        print("\n" + "=" * 60)
        print("详细测试报告汇总")
        print("=" * 60)
        for name, info in detailed_results.items():
            if isinstance(info, dict) and "status" in info:
                icon = "✅" if info["status"] == "通过" else "❌"
                print(f"  {icon} {name}: {info['status']}")

        browser.close()


def _analyze_page(page, name):
    body_text = page.locator("body").inner_text()
    has_content = len(body_text.strip()) > 50

    headings = page.locator("h1, h2, h3").all()
    heading_texts = []
    for h in headings[:10]:
        try:
            heading_texts.append(h.inner_text().strip()[:50])
        except Exception:
            pass

    links = page.locator("a").count()
    buttons = page.locator("button").count()

    return {
        "status": "通过" if has_content else "内容不足",
        "has_content": has_content,
        "content_length": len(body_text),
        "heading_count": len(heading_texts),
        "headings": heading_texts,
        "link_count": links,
        "button_count": buttons
    }


def _print_page_info(name, info):
    print(f"  状态: {info['status']}")
    print(f"  内容长度: {info['content_length']} 字符")
    print(f"  标题元素: {info['heading_count']} 个")
    print(f"  链接数: {info.get('link_count', 'N/A')}")
    print(f"  按钮数: {info.get('button_count', 'N/A')}")
    if info.get("headings"):
        for h in info["headings"][:5]:
            print(f"    - {h}")


def _test_navigation_clicks(page):
    results = {}
    routes = [
        ("/", "首页"),
        ("#/agents", "代理"),
        ("#/skills", "技能"),
        ("#/store", "商店"),
        ("#/projects", "项目"),
        ("#/assistant", "助手"),
        ("#/settings", "设置")
    ]

    for route, name in routes:
        try:
            url = f"{BASE_URL}/{route}" if route.startswith("#") else f"{BASE_URL}{route}"
            page.goto(url)
            page.wait_for_load_state("networkidle")
            page.wait_for_timeout(800)

            body_text = page.locator("body").inner_text()
            ok = len(body_text.strip()) > 50
            results[name] = "✅ 正常" if ok else "❌ 内容不足"
            print(f"  {name} ({route}): {'正常' if ok else '内容不足'}")
        except Exception as e:
            results[name] = f"❌ 错误: {e}"
            print(f"  {name} ({route}): 错误 - {e}")

    return results


if __name__ == "__main__":
    test_detailed()
