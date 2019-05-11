## Run the server

```
make
./bin/muser --addr 127.0.0.1:8000 --region us-west-2 --table dev.muser.codemk8
```

## Send request by curl 

```bash
# register a user 
$ curl -X POST -H "Content-Type: application/json" -d '{"user_name": "test_user", "password": "secret"}' http://localhost:8000/v1/user/register
```

```bash
# authorize a user
$ curl -X GET --user test_user:password  http://localhost:8000/v1/user/auth
# Get Wrong user name or password

$ curl -X GET --user test_user:secret  http://localhost:8000/v1/user/auth
```

```bash
# Change password
$ curl -X POST -H "Content-Type: application/json" -d '{"user_name": "test_user", "password": "secret", "new_password":"secret2"}' http://localhost:8000/v1/user/update
# Change pack
$ curl -X POST -H "Content-Type: application/json" -d '{"user_name": "test_user", "password": "secret2", "new_password":"secret"}' http://localhost:8000/v1/user/update
```