# Chirpy

```
                                  ,-,
                                 ( O)>
                                  `-'
                                 /(_)                                // ||\
                               //  || \
                              //   ||  \
                             //    ||   \
                            //     ||    \
                           //      ||     \
                          //       ||      \
                         //        ||       \
                        //         ||        \
                       //          ||         \
                      //___________||__________\
                     //            ||           \
                    ((             ||            ))
                     \___________||___________//
                                  ||
                                  ||
                                  ||
                                  ||
                                  ||
                                 /||                                /_||_                                / O||O                               /  ====                               /________                              \ (@)(@) /
                               \  /\  /
                                \ VV /
                                 \__/
```

A simple, Twitter-like API backend built in Go.

## Features

*   **Users:** Create and manage user accounts.
*   **Chirps:** Post short messages (up to 140 characters), view, and delete them.
*   **Authentication:** Uses JWT for secure API access.
*   **Profanity Filter:** Automatically censors certain words in chirps.
*   **Database:** Uses PostgreSQL to store data.
*   **Admin:** Includes endpoints for server health, metrics, and data reset.

#### Token Expiry
*   Access Tokens (JWTs) are short-lived and expire after 1 hour.
*   Refresh Tokens are long-lived and expire after 60 days if not used. They can be revoked via the `/api/revoke` endpoint.

## API Endpoints

The main API endpoints include:

*   `POST /api/login`: User login
*   `POST /api/refresh`: Refresh JWT token
*   `POST /api/revoke`: Revoke JWT token
*   `GET /api/chirps`: Retrieve chirps (can be sorted and filtered by author)
*   `POST /api/chirps`: Create a new chirp
*   `GET /api/chirps/{chirpID}`: Get a specific chirp
*   `DELETE /api/chirps/{chirpID}`: Delete a chirp
*   `POST /api/users`: Create a new user
*   `PUT /api/users`: Update user information
*   `POST /api/polka/webhooks`: Webhook for external service integration (user upgrades)
*   `GET /api/healthz`: Server readiness check
*   `GET /admin/metrics`: View application metrics (Shows how many times the Chirpy file server at /app/ has been visited since the server started).
*   `POST /admin/reset`: Reset application data (metrics)

## Database Schema

Chirpy uses a PostgreSQL database with the following main tables:

### `users`

Stores user account information.

| Column          | Type      | Constraints                               | Description                                  |
|-----------------|-----------|-------------------------------------------|----------------------------------------------|
| `id`            | UUID      | PRIMARY KEY                               | Unique identifier for the user               |
| `created_at`    | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP       | Timestamp of user creation                   |
| `updated_at`    | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP       | Timestamp of last user update                |
| `email`         | TEXT      | NOT NULL, UNIQUE                          | User's email address                         |
| `hashed_password` | TEXT      | NOT NULL                                  | Hashed password for the user                 |
| `is_chirpy_red` | BOOL      | NOT NULL, DEFAULT false                   | Indicates if the user has "Chirpy Red" status |

### `chirps`

Stores the chirps (posts) made by users.

| Column       | Type      | Constraints                               | Description                                     |
|--------------|-----------|-------------------------------------------|-------------------------------------------------|
| `id`         | UUID      | PRIMARY KEY                               | Unique identifier for the chirp                 |
| `created_at` | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP       | Timestamp of chirp creation                     |
| `updated_at` | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP       | Timestamp of last chirp update                  |
| `body`       | TEXT      | NOT NULL                                  | Content of the chirp (max 140 chars enforced by API) |
| `user_id`    | UUID      | NOT NULL, FOREIGN KEY (users.id) ON DELETE CASCADE | ID of the user who posted the chirp             |

### `refresh_tokens`

Stores refresh tokens used to obtain new access tokens.

| Column       | Type      | Constraints                               | Description                               |
|--------------|-----------|-------------------------------------------|-------------------------------------------|
| `token`      | TEXT      | PRIMARY KEY                               | The refresh token string                  |
| `created_at` | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP       | Timestamp of token creation               |
| `updated_at` | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP       | Timestamp of last token update            |
| `user_id`    | UUID      | NOT NULL, FOREIGN KEY (users.id) ON DELETE CASCADE | ID of the user this token belongs to      |
| `expires_at` | TIMESTAMP | NOT NULL, DEFAULT (creation + 60 days)    | Timestamp when the token expires          |
| `revoked_at` | TIMESTAMP | NULL                                      | Timestamp if the token has been revoked   |

---

Powered by Go!
