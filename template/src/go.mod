module template

go 1.14

require (
	cloud.google.com/go v0.62.0 // indirect
	github.com/DATA-DOG/go-txdb v0.1.3
	github.com/bxcodec/faker/v3 v3.5.0
	github.com/friendsofgo/errors v0.9.2
	github.com/go-playground/validator/v10 v10.3.0
	github.com/gofrs/uuid v3.3.0+incompatible // indirect
	github.com/golang-migrate/migrate/v4 v4.12.1
	github.com/graph-gophers/dataloader v5.0.0+incompatible
	github.com/graph-gophers/graphql-go v0.0.0-20200622220639-c1d9693c95a6
	github.com/kat-co/vala v0.0.0-20170210184112-42e1d8b61f12
	github.com/lib/pq v1.7.0
	github.com/scott-rc/core v0.0.0-20200731062322-ee14506b2064
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.6.1
	github.com/volatiletech/null/v8 v8.1.0
	github.com/volatiletech/randomize v0.0.1
	github.com/volatiletech/sqlboiler/v4 v4.2.0
	github.com/volatiletech/strmangle v0.0.1
	golang.org/x/crypto v0.0.0-20200709230013-948cd5f35899
	google.golang.org/genproto v0.0.0-20200731012542-8145dea6a485 // indirect
	google.golang.org/grpc v1.31.0 // indirect
)

//replace github.com/scott-rc/core => ../../
