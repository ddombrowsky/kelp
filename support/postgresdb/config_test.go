package postgresdb

import (
	"database/sql"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSSLMode(t *testing.T) {
	assert.Equal(t, "disable", (&Config{SSLEnable: false}).GetSSLMode())
	assert.Equal(t, "require", (&Config{SSLEnable: true}).GetSSLMode())
}

func TestMakeConnectString(t *testing.T) {
	testCases := []struct {
		name   string
		config Config
		want   string
	}{
		{
			name:   "defaults",
			config: Config{},
			want:   "host=localhost port=5432 sslmode=disable dbname=kelp",
		}, {
			name: "all fields set",
			config: Config{
				Host:      "db.example.com",
				Port:      6543,
				DbName:    "trades",
				User:      "kelp",
				Password:  "secret",
				SSLEnable: true,
			},
			want: "host=db.example.com port=6543 sslmode=require user=kelp password=secret dbname=trades",
		},
	}

	for _, k := range testCases {
		t.Run(k.name, func(t *testing.T) {
			assert.Equal(t, k.want, k.config.MakeConnectString())
		})
	}
}

// the postgres driver validates sslmode only after the TCP connection is made, so this runs a listener that accepts
// and immediately closes connections: an unsupported sslmode is reported as such, anything else fails with a connection error
func TestSSLModeAcceptedByDriver(t *testing.T) {
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if !assert.NoError(t, e) {
		return
	}
	defer listener.Close()
	go func() {
		for {
			conn, e := listener.Accept()
			if e != nil {
				return
			}
			conn.Close()
		}
	}()
	port := uint16(listener.Addr().(*net.TCPAddr).Port)

	for _, sslEnable := range []bool{false, true} {
		c := &Config{Host: "127.0.0.1", Port: port, SSLEnable: sslEnable}
		db, e := sql.Open("postgres", c.MakeConnectString())
		if !assert.NoError(t, e) {
			continue
		}

		e = db.Ping()
		db.Close()
		if assert.Error(t, e) {
			assert.NotContains(t, e.Error(), "sslmode", "sslmode=%s was rejected by the driver", c.GetSSLMode())
		}
	}
}
