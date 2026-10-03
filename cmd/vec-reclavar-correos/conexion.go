package main

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

type destinoConexion struct {
	Host      string `json:"host"`
	Puerto    uint16 `json:"puerto"`
	Usuario   string `json:"usuario"`
	SSLMode   string `json:"sslmode"`
	ClonLocal bool   `json:"clon_local"`
	CAHuella  string `json:"ca_sha256"`
}

func huellaConexion(base string, d destinoConexion) string {
	b, _ := json.Marshal(struct {
		Base     string          `json:"base"`
		Conexion destinoConexion `json:"conexion"`
	}{base, d})
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func configuracionConexion(b []byte, p plan, ca []byte) (*pgx.ConnConfig, error) {
	for _, nombre := range []string{"PGHOST", "PGPORT", "PGDATABASE", "PGUSER", "PGPASSWORD", "PGPASSFILE", "PGAPPNAME", "PGCONNECT_TIMEOUT", "PGSSLMODE", "PGSSLKEY", "PGSSLCERT", "PGSSLSNI", "PGSSLROOTCERT", "PGSSLPASSWORD", "PGSSLNEGOTIATION", "PGTARGETSESSIONATTRS", "PGSERVICE", "PGSERVICEFILE", "PGTZ", "PGOPTIONS", "PGMINPROTOCOLVERSION", "PGMAXPROTOCOLVERSION", "PGCHANNELBINDING", "PGREQUIREAUTH"} {
		if _, presente := os.LookupEnv(nombre); presente {
			return nil, errEntrada
		}
	}
	u, err := url.Parse(strings.TrimSpace(string(b)))
	d := p.Conexion
	if err != nil || (u.Scheme != "postgresql" && u.Scheme != "postgres") || d.Host == "" || len(d.Host) > 253 || d.Puerto == 0 || u.Hostname() != d.Host || u.Port() != strconv.FormatUint(uint64(d.Puerto), 10) || u.User == nil || u.User.Username() != d.Usuario || d.Usuario == "" || u.Path != "/"+p.Base || u.Fragment != "" || u.RawQuery == "" || p.ConexionHuella != huellaConexion(p.Base, d) {
		return nil, errEntrada
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil || len(q) != 1 || len(q["sslmode"]) != 1 || q.Get("sslmode") != d.SSLMode {
		return nil, errEntrada
	}
	var tlsConfig *tls.Config
	switch d.SSLMode {
	case "disable":
		if !d.ClonLocal || d.Host != "127.0.0.1" || len(ca) != 0 || d.CAHuella != "" {
			return nil, errEntrada
		}
	case "verify-full":
		h := sha256.Sum256(ca)
		if len(ca) == 0 || !digestValido.MatchString(d.CAHuella) || hex.EncodeToString(h[:]) != d.CAHuella {
			return nil, errEntrada
		}
		roots := x509.NewCertPool()
		if !roots.AppendCertsFromPEM(ca) {
			return nil, errEntrada
		}
		tlsConfig = &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots, ServerName: d.Host}
	default:
		return nil, errEntrada
	}
	// ParseConfig necesita inicializar campos privados de pgx. Se neutraliza
	// SSL antes de llamarlo: jamás carga PGSSLROOTCERT/PGSSLKEY implícitos.
	q.Set("sslmode", "disable")
	q.Set("passfile", "/dev/null")
	u.RawQuery = q.Encode()
	cfg, err := pgx.ParseConfig(u.String())
	if err != nil || cfg.Host != d.Host || cfg.Database != p.Base {
		return nil, errEntrada
	}
	password, _ := u.User.Password()
	cfg.Password = password
	cfg.TLSConfig = tlsConfig
	cfg.Fallbacks = nil
	cfg.RuntimeParams = map[string]string{"application_name": "vec-reclavar-correos", "search_path": "pg_catalog", "statement_timeout": "30000", "lock_timeout": "5000", "timezone": "UTC"}
	return cfg, nil
}
