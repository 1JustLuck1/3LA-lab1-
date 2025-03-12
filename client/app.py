from datetime import datetime, timedelta
from flask import Flask, render_template, request, redirect, url_for, session, flash
import requests
import jwt
import os

app = Flask(__name__)
app.secret_key = os.urandom(32).hex()

BASE_URL = "http://server:8080"

def get_user_id_from_token(token):
    try:
        decoded = jwt.decode(token, "bq3X7Z8k9y2A1B4C5D6E7F8G9H0I1J2K3L4M5N6O7P8Q9R0S", algorithms=["HS256"])
        return decoded.get("user_id")
    except:
        return None

#Главная страница
@app.route("/")
def index():
    if "token" in session:
        return redirect(url_for("orders"))
    return redirect(url_for("login"))

#Регистрация
@app.route("/register", methods=["GET", "POST"])
def register():
    if request.method == "POST":
        username = request.form["username"]
        password = request.form["password"]
        response = requests.post(
            f"{BASE_URL}/register",
            json={"username": username, "password": password},
        )
        if response.status_code == 200:
            flash("Регистрация успешна! Пожалуйста, войдите.")
            return redirect(url_for("login"))
        else:
            flash("Ошибка регистрации: " + response.json().get("error", "Unknown error"))
    return render_template("register.html")

#Авторизация
@app.route("/login", methods=["GET", "POST"])
def login():
    if request.method == "POST":
        username = request.form["username"]
        password = request.form["password"]
        response = requests.post(
            f"{BASE_URL}/login",
            json={"username": username, "password": password},
        )
        if response.status_code == 200:
            session["token"] = response.json().get("token")
            return redirect(url_for("orders"))
        else:
            flash("Ошибка авторизации: Неверный логин или пароль")
    return render_template("login.html")

#Заказы пользователя
@app.route("/orders")
def orders():
    if "token" not in session:
        return redirect(url_for("login"))

    headers = {"Authorization": session["token"]}
    try:
        user_id = get_user_id_from_token(session["token"])
        response = requests.get(
            f"{BASE_URL}/auth/orders?user_id={user_id}",
            headers=headers,
        )
        response.raise_for_status()
        orders = response.json()
        return render_template("orders.html", orders=orders)
    except requests.exceptions.RequestException as e:
        print("Request failed:", e)
        if response:
            try:
                error_message = response.json().get("error", "Unknown error")
            except ValueError:
                error_message = response.text
        else:
            error_message = str(e)
        # flash(f"Ошибка получения заказов: {error_message}")
        return redirect(url_for("index"))

#Создание нового заказа
@app.route("/create_order", methods=["GET", "POST"])
def create_order():
    if "token" not in session:
        return redirect(url_for("login"))
    
    if request.method == "POST":
        book_name = str(request.form["booksDropdown"])
        quantity = int(request.form["quantity"])

        headers = {"Authorization": session["token"]}
        try:
            response = requests.post(
                f"{BASE_URL}/auth/createOrder",
                json={"bookName":book_name,"quantity": quantity},
                headers=headers,
            )
            response.raise_for_status()
            flash("Заказ успешно создан!")
        except requests.exceptions.RequestException as e:
            print("Request failed:", e)
            if response:
                try:
                    error_message = response.json().get("error", "Unknown error")
                except ValueError:
                    error_message = response.text
            else:
                error_message = str(e)
            flash(f"Ошибка создания заказа: {error_message}")
    return render_template("create_order.html")

#Детали конкретного заказа
@app.route("/order_detail/<int:order_id>")
def order_detail(order_id):
    if "token" not in session:
        return redirect(url_for("login"))

    headers = {"Authorization": session["token"]}
    try:
        response = requests.get(
            f"{BASE_URL}/auth/orderDetail/{order_id}",  # Проверьте этот URL
            headers=headers,
        )
        if response.status_code == 200:
            OrderItems = response.json()
            return render_template("order_detail.html",OrderItems=OrderItems, order_id=order_id)
        else:
            flash("Ошибка получения инфомарции о заказе: " + response.json().get("error", "Unknown error"))
    except requests.exceptions.RequestException as e:
        print("Request failed:", e)
        if response:
            try:
                error_message = response.json().get("error", "Unknown error")
            except ValueError:
                error_message = response.text
        else:
            error_message = str(e)
        flash(f"Ошибка получения инфомарции о заказе: {error_message}")
    return redirect(url_for("orders"))

#Удаление существующего заказа
@app.route("/delete_order/<int:order_id>")
def delete_order(order_id):
    if "token" not in session:
        return redirect(url_for("login"))

    headers = {"Authorization": session["token"]}
    try:
        response = requests.delete(
            f"{BASE_URL}/auth/deleteOrder/{order_id}",
            headers=headers,
        )
        if response.status_code == 200:
            flash("Заказ успешно удалён!")
        else:
            flash("Ошибка удаления заказа: " + response.json().get("error", "Unknown error"))
    except requests.exceptions.RequestException as e:
        print("Request failed:", e)
        if response:
            try:
                error_message = response.json().get("error", "Unknown error")
            except ValueError:
                error_message = response.text
        else:
            error_message = str(e)
        flash(f"Ошибка удаления заказа: {error_message}")
    return redirect(url_for("orders"))

#Выход
@app.route("/logout")
def logout():
    session.pop("token", None)
    return redirect(url_for("index"))

#Проверка токена
@app.before_request
def check_token():
    if request.endpoint not in ["login", "register", "static"]:
        token = session.get("token")
        if not token or is_token_expired(token):
            return redirect(url_for("login"))

def is_token_expired(token):
    try:
        decoded = jwt.decode(token, "bq3X7Z8k9y2A1B4C5D6E7F8G9H0I1J2K3L4M5N6O7P8Q9R0S", algorithms=["HS256"])
        expiration_time = decoded["exp"]
        return datetime.utcnow() > datetime.fromtimestamp(expiration_time)
    except jwt.ExpiredSignatureError:
        return True
    except jwt.InvalidTokenError:
        return True

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=5000, debug=True)