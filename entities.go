package storage

const (
	// EntityOpenIDSessions provides the name of the entity to use in order to
	// create, read, update and delete OpenID Sessions.
	EntityOpenIDSessions = "openid_connect_session"

	// EntityAccessTokens provides the name of the entity to use in order to
	// create, read, update and delete Access Token sessions.
	EntityAccessTokens = "access_token"

	// EntityRefreshTokens provides the name of the entity to use in order to
	// create, read, update and delete Refresh Token sessions.
	EntityRefreshTokens = "refresh_token"

	// EntityAuthorizationCodes provides the name of the entity to use in order
	// to create, read, update and delete Authorization Code sessions.
	EntityAuthorizationCodes = "authorization_code"

	// EntityPKCESessions provides the name of the entity to use in order to
	// create, read, update and delete Proof Key for Code Exchange sessions.
	EntityPKCESessions = "pkce_session"

	// EntityJtiDenylist provides the name of the entity to use in order to
	// track and deny.
	EntityJtiDenylist = "jti_deny_list"

	// EntityClients provides the name of the entity to use in order to create,
	// read, update and delete Clients.
	EntityClients = "client"

	// EntityUsers provides the name of the entity to use in order to create,
	// read, update and delete Users.
	EntityUsers = "user"
)
