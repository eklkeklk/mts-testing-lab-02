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
