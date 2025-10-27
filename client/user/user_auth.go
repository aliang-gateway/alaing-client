package user

var hy2Username = "lisi"
var hy2Password = "IW6gUxtuG46FURELO08p9L9I3GtHtfh1"
var userUUID = ""

func SetUserUUID(uuid string) {
	userUUID = uuid
}

func GetUserUUID() string {
	return userUUID
}

func SetUsername(username string) {
	hy2Username = username
}

func SetPassword(password string) {
	hy2Password = password
}

func GetUsername() string {
	return hy2Username
}

func GetPassword() string {
	return hy2Password
}
