// Package jwtauth — этот файл оставлен пустым после Phase 4.
//
// Раньше здесь жил AES-256-GCM "SecretBox" для шифрования peer-private-keys
// при хранении в БД. После перехода на client-side keygen (Phase 4) сервер
// больше НЕ ВИДИТ приватных ключей — шифровать нечего, файл deprecated.
//
// Не удаляем, чтобы сохранить путь импорта без массовых rebase'ов.
package jwtauth
