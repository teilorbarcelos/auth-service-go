package domainerr

import "errors"

var (
	ErrUserNotFound          = errors.New("usuário não encontrado")
	ErrInvalidCredentials     = errors.New("credenciais inválidas")
	ErrAccountDisabled       = errors.New("conta desativada ou removida")
	ErrAuthNotConfigured     = errors.New("autenticação não configurada para este usuário")
	ErrUnauthorized          = errors.New("não autorizado")
	ErrSessionCreationFailed = errors.New("falha ao criar sessão")
	ErrInternal              = errors.New("erro interno do servidor")
	ErrInvalidToken          = errors.New("token inválido")
	ErrTokenExpired          = errors.New("token expirado")
)
