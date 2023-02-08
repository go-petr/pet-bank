## Description

#### Features

This bank service service provides APIs for the frontend to do following things:
1. Create and login users
2. Create, get and list users own accounts of different currencies
3. Transfer money between two accounts within a transaction with recording all balance changes in account entries

#### Authorization rules for logged-in user

1. only create an account for him/herself
2. only get accounts that he/she owns
3. only list accounts that belong to him/her
4. only send money from his/her own account
5. only refresh his/own access token

## How to run

#### Development

```
docker-compose -f deployments/docker-compose.yaml up
```

#### Production (AWS)

##### Setup 

Set up GitHub Actions repository secrets
1. AWS_ACCESS_KEY_ID
2. AWS_SECRET_ACCESS_KEY
3. DB_DRIVER
4. DB_SOURCE

TODO

