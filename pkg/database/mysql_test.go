package database

import (
	"testing"

	mysqldriver "github.com/go-sql-driver/mysql"
)

func TestBuildMySQLDSNPreservesLegacyLocalDefault(t *testing.T) {
	raw, err := buildMySQLDSN(&MySQLConfig{Host: "127.0.0.1:3306", Username: "user", Password: "pass", Database: "qs"})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := mysqldriver.ParseDSN(raw)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Loc.String() != "Local" || !parsed.ParseTime || !parsed.MultiStatements {
		t.Fatalf("parsed=%+v", parsed)
	}
}

func TestBuildMySQLDSNAppliesLocationAndSessionTimeZone(t *testing.T) {
	raw, err := buildMySQLDSN(&MySQLConfig{
		Host: "127.0.0.1:3306", Username: "user", Password: "pass", Database: "qs",
		Location: "Asia/Shanghai", SessionTimeZone: "+08:00",
	})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := mysqldriver.ParseDSN(raw)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Loc.String() != "Asia/Shanghai" {
		t.Fatalf("location=%s", parsed.Loc)
	}
	if got := parsed.Params["time_zone"]; got != "'+08:00'" {
		t.Fatalf("time_zone=%q", got)
	}
}

func TestBuildMySQLDSNRejectsUnknownLocation(t *testing.T) {
	if _, err := buildMySQLDSN(&MySQLConfig{Location: "Mars/Olympus"}); err == nil {
		t.Fatal("expected invalid location to fail")
	}
}
