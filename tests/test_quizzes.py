import requests

def test_create_quiz(api_url, admin_token, get_auth_headers, test_class_id):
    payload = {
        "title": "Test Quiz API",
        "class_id": test_class_id,
        "passing_score": 70,
        "max_attempts": 2,
        "is_published": True
    }
    
    res = requests.post(f"{api_url}/quizzes", headers=get_auth_headers(admin_token), json=payload)
    assert res.status_code == 201, f"Create quiz failed: {res.text}"
    
    data = res.json()
    assert data["title"] == "Test Quiz API"
    assert data["passing_score"] == 70