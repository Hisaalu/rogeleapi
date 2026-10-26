import requests

def test_register_learner(api_url, generate_random_email, test_class_id):
    email = generate_random_email()
    payload = {
        "first_name": "Test",
        "last_name": "User",
        "email": email,
        "phone": "+256700000000",
        "class_id": test_class_id,
        "password": "Password123!",
        "role": "learner"
    }
    
    res = requests.post(f"{api_url}/auth/register", json=payload)
    assert res.status_code == 201, f"Register failed: {res.text}"
    
    data = res.json()
    assert "access_token" in data
    assert data["user"]["email"] == email

def test_register_duplicate_email(api_url, generate_random_email, test_class_id):
    email = generate_random_email()
    payload = {
        "first_name": "Test", "last_name": "User", "email": email,
        "phone": "+256700000000", "class_id": test_class_id,
        "password": "Password123!", "role": "learner"
    }
    
    requests.post(f"{api_url}/auth/register", json=payload)
    res = requests.post(f"{api_url}/auth/register", json=payload)
    
    assert res.status_code == 409, "Duplicate email should return 409 Conflict"

def test_login_wrong_password(api_url):
    res = requests.post(f"{api_url}/auth/login", json={
        "email": "",
        "password": ""
    })
    assert res.status_code == 401

def test_no_token_returns_401(api_url):
    res = requests.get(f"{api_url}/lessons")
    assert res.status_code == 401, "Missing token should return 401"