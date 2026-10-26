import os
import pytest
import requests
from uuid import uuid4
from dotenv import load_dotenv

root_dir = os.path.dirname(os.path.dirname(os.path.dirname(__file__)))
load_dotenv(dotenv_path=os.path.join(root_dir, ".env"))

BASE_URL = os.getenv("API_BASE_URL", "http://localhost:8081")
API_URL = f"{BASE_URL}/api/v1"

@pytest.fixture(scope="session")
def api_url():
    return API_URL

@pytest.fixture(scope="session")
def get_auth_headers():
    def _get_headers(token):
        return {"Authorization": f"Bearer {token}", "Content-Type": "application/json"}
    return _get_headers

@pytest.fixture(scope="session")
def generate_random_email():
    def _generate():
        return f"test_{uuid4().hex[:8]}@example.com"
    return _generate

@pytest.fixture(scope="session")
def admin_token():
    """Logs in as an admin and returns the token."""
    admin_email = os.getenv("TEST_ADMIN_EMAIL", "")
    admin_password = os.getenv("TEST_ADMIN_PASSWORD", "")

    res = requests.post(f"{API_URL}/auth/login", json={
        "email": admin_email,
        "password": admin_password
    })
    
    if res.status_code != 200:
        pytest.skip(f"Could not log in as admin. Check TEST_ADMIN_EMAIL/PASSWORD in .env. Response: {res.text}")
    
    return res.json().get("access_token")

@pytest.fixture(scope="session")
def test_class_id(admin_token, get_auth_headers):
    """Creates a test class to use across tests."""
    res = requests.post(
        f"{API_URL}/classes",
        headers=get_auth_headers(admin_token),
        json={"name": f"Test Class {uuid4().hex[:4]}", "level": "Test", "is_active": True}
    )
    if res.status_code not in (200, 201):
        pytest.skip(f"Could not create test class: {res.text}")
    return res.json().get("id")