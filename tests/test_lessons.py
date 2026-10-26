import requests

def test_create_lesson_as_admin(api_url, admin_token, get_auth_headers, test_class_id):
    payload = {
        "title": "Test Lesson API",
        "content": "This is a test content",
        "class_id": test_class_id,
        "is_published": True
    }
    
    res = requests.post(f"{api_url}/lessons", headers=get_auth_headers(admin_token), json=payload)
    assert res.status_code == 201, f"Create lesson failed: {res.text}"
    
    data = res.json()
    assert data["title"] == "Test Lesson API"

def test_list_lessons(api_url, admin_token, get_auth_headers):
    res = requests.get(f"{api_url}/lessons", headers=get_auth_headers(admin_token))
    assert res.status_code == 200
    assert isinstance(res.json(), list)

def test_learner_cannot_create_lesson(api_url, generate_random_email, test_class_id, get_auth_headers):
    email = generate_random_email()
    reg_res = requests.post(f"{api_url}/auth/register", json={
        "first_name": "Learner", "last_name": "Test", "email": email,
        "phone": "+256700000000", "class_id": test_class_id,
        "password": "Password123!", "role": "learner"
    })
    learner_token = reg_res.json()["access_token"]
    
    payload = {"title": "Hack Lesson", "class_id": test_class_id, "is_published": True}
    res = requests.post(f"{api_url}/lessons", headers=get_auth_headers(learner_token), json=payload)
    
    assert res.status_code == 403, "Learner should not be able to create lessons"