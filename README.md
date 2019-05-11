## Run the server


## Send request by curl 

```bash
# register a user 
$ curl -X POST -H "Content-Type: application/json" -d '{"user_name": "test_user", "password": "secret"}' http://localhost:8000/user/register
```

```bash
# authorize a user
$ curl -X GET --user test_user:password  http://localhost:8000/user/auth
# Get Wrong user name or password

$ curl -X GET --user test_user:secret  http://localhost:8000/user/auth
```

```bash
$ curl -X POST http://localhost:8000/user/update
```