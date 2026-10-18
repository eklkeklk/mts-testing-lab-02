# Домашняя работа №2

## Запуск

Из корня репозитория:

```sh
go test ./... -v
```

## Сценарии

| Тест | Что проверяет |
| --- | --- |
| `TestListPosts` | список постов с лимитом 3, идентификаторы элементов и метаданные пагинации |
| `TestGetUserByID` | пользователь /users/1, логин, имя и email |
| `TestPostCreatePatchGet` | создание поста, извлечение id, изменение title, чтение того же поста и сохранение body и user_id |
| `TestLoginTokenAndCurrentUser` | получение access_token при входе и передача Bearer в /auth/me, проверка пользователя |
| `TestAuthMeRejectsMissingAndInvalidToken` | отказ защищённого API без токена и с неверным токеном |
| `TestGraphQLHelloQuery` | данные Hello и отсутствие GraphQL errors |
| `TestGraphQLCreatePersonMutation` | id, name, age созданного человека и отсутствие errors |
| `TestGraphQLInvalidFieldReturnsErrors` | Дополнительный негативный сценарий: ошибка валидации неизвестного GraphQL поля |

# Домашняя работа №3

Контейнер запускает существующий `TestGraphQLHelloQuery`: тест отправляет GraphQL запрос к публичному API и проверяет ответ

Команды сборки и запуска контейнера:

```sh
docker build -t api-test .
docker run --rm api-test
```

Тест запускается автоматически при старте контейнера. Код завершения контейнера равен коду завершения `go test` (0 при успехе, ненулевой при ошибке)

Конфигурация GH Actions находится в [`.github/workflows/ci.yml`](.github/workflows/ci.yml). При каждом пуше она получает код из репозитория, собирает образ и запускает на нем тест. Результат выполнения после отправки коммита доступен на [странице workflow](https://github.com/eklkeklk/mts-testing-lab-02/actions/workflows/ci.yml)
