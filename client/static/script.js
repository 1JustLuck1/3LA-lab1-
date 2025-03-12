$(document).ready(function() {
    $.getJSON("http://localhost:8080/books", function(data) {
        var options = '';
        $.each(data, function(key, book) {
            options += '<option value="' + book.Name + '">' + book.Name + ' (' + book.Genre + ') - $' + book.Price.toFixed(2) + '</option>';
        });
        $("#booksDropdown").html(options);
    });
});

// Проверка, истёк ли токен
function isTokenExpired(token) {
    if (!token) return true;

    try {
        const payload = JSON.parse(atob(token.split('.')[1]));  // Декодируем полезную нагрузку токена
        const expirationTime = payload.exp * 1000;  // Время истечения в миллисекундах
        return Date.now() > expirationTime;  // Проверяем, истёк ли токен
    } catch (error) {
        console.error("Error decoding token:", error);
        return true;
    }
}
// Перехватчик ошибок для всех запросов
axios.interceptors.response.use(
    response => response,
    error => {
        if (error.response && error.response.status === 401) {
            localStorage.removeItem("token");  // Удаляем токен из локального хранилища
            window.location.href = "/login";  // Перенаправляем на страницу авторизации
        }
        return Promise.reject(error);
    }
);