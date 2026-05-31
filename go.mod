module raki

go 1.25.9

replace raki/adminapi => ./packages/soju-admin-api

require raki/adminapi v0.0.0-00010101000000-000000000000

require (
	golang.org/x/time v0.0.0-20220722155302-e5dcc9cfc0b9 // indirect
	gopkg.in/irc.v4 v4.0.0 // indirect
)

replace raki/client => ./packages/raki-client
