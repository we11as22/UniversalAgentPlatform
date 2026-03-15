from uap_common import AppConfig, create_app


def test_create_app_health_route() -> None:
    app = create_app(AppConfig(service_name="unit-test"))
    assert app.title == "unit-test"

