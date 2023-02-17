# Settings API
This is a simple Go REST API for a `Settings` serivce, built on `Echo` framework.

## Building & Running
To build and run the application, run the following commands:
```bash
docker-compose build
docker-compose up -d
```

## Auth
The API uses JWT for authentication. To generate a token, run call sign-in endpoint with the following credentials:
```
username: admin
password: SabziPolo
```

## Queries

### Hello
```
curl -X GET http://localhost:8090/
```
```json
{
  "message": "Hello!"
}
```
### Sign-in
```
curl --request POST \
  --url http://localhost:8090/signin \
  --header 'Content-Type: application/json' \
  --data '{
	"username": "admin",
	"password": "SabziPolo"
}'
```
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2NzY4OTM3ODUsInVzZXJuYW1lIjoiYWRtaW4ifQ.dh-5OPn_njxoGhpaBeys8ncCnvEPY5UCEx2JZNl7xbo"
}
```

### Create Setting
```
curl --request POST \
  --url http://localhost:8090/settings \
  --header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2NzY4OTM3ODUsInVzZXJuYW1lIjoiYWRtaW4ifQ.dh-5OPn_njxoGhpaBeys8ncCnvEPY5UCEx2JZNl7xbo' \
  --header 'Content-Type: application/json' \
  --data '{
	"key": "new-setting",
	"value": "value for the new setting",
	"ttl": 6000
}'
```
```json
{
  "key": "new-setting",
  "value": "value for the new setting",
  "ttl": 6000,
  "created_at": "0001-01-01T00:00:00Z",
  "updated_at": "0001-01-01T00:00:00Z"
}
```

### Updating Setting
```
curl --request PUT \
  --url http://localhost:8090/settings/aa \
  --header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2NzY4OTM3ODUsInVzZXJuYW1lIjoiYWRtaW4ifQ.dh-5OPn_njxoGhpaBeys8ncCnvEPY5UCEx2JZNl7xbo' \
  --header 'Content-Type: application/json' \
  --data '{
	"key": "new-setting",
	"value": "this is a modified value",
	"ttl": 1000
}'
```
```json
{
  "key": "new-setting",
  "value": "this is a modified value",
  "ttl": 1000,
  "created_at": "0001-01-01T00:00:00Z",
  "updated_at": "0001-01-01T00:00:00Z"
}
```

### Get All Settings
```
curl --request GET \
  --url http://localhost:8090/settings \
  --header 'Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2NzY4OTM3ODUsInVzZXJuYW1lIjoiYWRtaW4ifQ.dh-5OPn_njxoGhpaBeys8ncCnvEPY5UCEx2JZNl7xbo' \
  --header 'Content-Type: application/json'
```
```json
[
  {
    "key": "new-setting",
    "value": "value for the new setting",
    "ttl": 956,
    "created_at": "2023-02-17T15:01:07.332995Z",
    "updated_at": "2023-02-17T15:01:07.332995Z"
  }
]
```

