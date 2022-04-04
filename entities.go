package storage

const (
	CollectionPrefix = "oauth2_"
	// EntityOpenIDSessions provides the name of the entity to use in order to
	// create, read, update and delete OpenID Sessions.
	EntityOpenIDSessions = CollectionPrefix + "openid_connect_session"

	// EntityAccessTokens provides the name of the entity to use in order to
	// create, read, update and delete Access Token sessions.
	EntityAccessTokens = CollectionPrefix + "access_token"

	// EntityRefreshTokens provides the name of the entity to use in order to
	// create, read, update and delete Refresh Token sessions.
	EntityRefreshTokens = CollectionPrefix + "refresh_token"

	// EntityAuthorizationCodes provides the name of the entity to use in order
	// to create, read, update and delete Authorization Code sessions.
	EntityAuthorizationCodes = CollectionPrefix + "authorization_code"

	// EntityPKCESessions provides the name of the entity to use in order to
	// create, read, update and delete Proof Key for Code Exchange sessions.
	EntityPKCESessions = CollectionPrefix + "pkce_session"

	// EntityJtiDenylist provides the name of the entity to use in order to
	// track and deny.
	EntityJtiDenylist = CollectionPrefix + "jti_deny_list"

	// EntityClients provides the name of the entity to use in order to create,
	// read, update and delete Clients.
	EntityClients = CollectionPrefix + "client"

	// EntityUsers provides the name of the entity to use in order to create,
	// read, update and delete Users.
	EntityUsers = CollectionPrefix + "user"
)
