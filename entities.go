package storage

const (
	// EntityOpenIDSessions provides the name of the entity to use in order to
	// create, read, update and delete OpenID Sessions.
	EntityOpenIDSessions = "openidconnectsession"

	// EntityAccessTokens provides the name of the entity to use in order to
	// create, read, update and delete Access Token sessions.
	EntityAccessTokens = "accesstoken"

	// EntityRefreshTokens provides the name of the entity to use in order to
	// create, read, update and delete Refresh Token sessions.
	EntityRefreshTokens = "refreshtoken"

	// EntityAuthorizationCodes provides the name of the entity to use in order
	// to create, read, update and delete Authorization Code sessions.
	EntityAuthorizationCodes = "authorizationcode"

	// EntityPKCESessions provides the name of the entity to use in order to
	// create, read, update and delete Proof Key for Code Exchange sessions.
	EntityPKCESessions = "pkcesession"

	// EntityJtiDenylist provides the name of the entity to use in order to
	// track and deny.
	EntityJtiDenylist = "jtidenylist"

	// EntityClients provides the name of the entity to use in order to create,
	// read, update and delete Clients.
	EntityClients = "client"

	// EntityUsers provides the name of the entity to use in order to create,
	// read, update and delete Users.
	EntityUsers = "user"
)
