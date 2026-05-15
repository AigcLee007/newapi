# Custom API Key Creation

Root users can optionally set a custom API key when creating a token. If the
field is left blank, the system keeps using the normal random key generator.

Regular users and non-root admins cannot set custom keys. The server rejects a
create request that includes a non-empty `key` unless the current user is root.

Rules:

- Custom keys may be sent with or without the `sk-` prefix.
- Keys are stored without the `sk-` prefix, matching existing token behavior.
- Length after removing `sk-` must be 16 to 128 characters.
- Allowed characters are `A-Z`, `a-z`, `0-9`, `_`, `-`, and `.`.
- Keys must be globally unique, including soft-deleted token records.
- Existing token edit flows do not change token keys.

Security notes:

- Do not use short, predictable, personal, or publicly known values.
- Do not put full keys in logs, screenshots, tickets, or public chats.
- Treat custom keys the same way as randomly generated API keys.
