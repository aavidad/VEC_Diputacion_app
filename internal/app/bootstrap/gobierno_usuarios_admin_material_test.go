package bootstrap

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"vec-diputacion-granada/internal/app/administracion"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
)

type relojGobiernoUsuariosPrueba struct{ ahora time.Time }

func (r relojGobiernoUsuariosPrueba) Ahora() time.Time { return r.ahora }

func configuracionGobiernoUsuariosPrueba(t *testing.T) (ConfiguracionMaterialUsuariosAdmin, relojGobiernoUsuariosPrueba) {
	t.Helper()
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "idempotencia"), 0700); err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(dir, "idempotencia", "configuracion.json")
	if err := os.WriteFile(ruta, configuracionIdempotenciaJSONPrueba(2, 1), 0600); err != nil {
		t.Fatal(err)
	}
	for _, g := range materialIdempotenciaDeterministaPrueba(2, 1).generaciones {
		for dominio, secreto := range map[string][]byte{"localizador": g.localizador.material[:], "huella-solicitud": g.huellaSolicitud.material[:]} {
			if err := os.WriteFile(rutaClaveIdempotenciaDesarrollo(dir, g.generacion, dominio), secreto, 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	reloj := relojGobiernoUsuariosPrueba{time.Date(2026, 10, 4, 8, 30, 0, 0, time.UTC)}
	base, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(nuevoDerivadorIdempotenciaPrueba(t, 2, 1), reloj.Ahora())
	if err != nil {
		t.Fatal(err)
	}
	defer borrarBytes(base.privada)
	defer borrarBytes(base.claveHMAC)
	semillaArchivo := filepath.Join(dir, "semilla-root-dev.bin")
	if err := os.WriteFile(semillaArchivo, base.privada.Seed(), 0600); err != nil {
		t.Fatal(err)
	}
	c := ConfiguracionMaterialUsuariosAdmin{DirectorioMaterial: dir, RutaConfiguracionHMAC: ruta, ArchivoSemillaRaiz: semillaArchivo, PrefijoEvidencia: "evidencia:firma:admin:usuarios:", Raiz: administracion.MaterialRaizPerfilesV3{ClaveID: base.claveID, Audiencia: audienciaAtestacionContratacionTemporalDesarrollo, Version: base.claveVersion, Publica: append(ed25519.PublicKey(nil), base.privada.Public().(ed25519.PublicKey)...), Estado: confianza.EstadoClaveAtestacionAutorizacionV3Activa, ValidaDesde: base.validaDesde, ValidaHasta: base.validaHasta}, Gobierno: administracion.GobiernoConfianzaPerfilesV3{Revision: base.configuracionRef, Secuencia: base.configuracionOrden, HuellaSHA256: base.configuracionHuella, PublicadaEn: base.publicadaEn, ExpiraEn: base.expiraEn}}
	for _, s := range []struct{ a, n string }{{administracion.AudienciaUsuariosListarV3, "listar"}, {administracion.AudienciaUsuariosConsultarV3, "consultar"}} {
		c.Entradas = append(c.Entradas, DescriptorClaveUsuariosAdmin{Audiencia: s.a, Dominio: "vec.admin.desarrollo.usuarios." + s.n, PrefijoClave: "clave:capacidad:admin:usuarios:" + s.n + ":", EmisorID: "emisor:admin:usuarios:desarrollo:v1", Version: 1, RevisionGobierno: 1, ValidaDesde: reloj.Ahora().Add(-time.Minute), ValidaHasta: reloj.Ahora().Add(time.Hour)})
	}
	return c, reloj
}

func TestGobiernoUsuariosMaterialReutilizaProveedorDosAudienciasYCierre(t *testing.T) {
	cfg, reloj := configuracionGobiernoUsuariosPrueba(t)
	m, err := PrepararMaterialUsuariosAdmin(context.Background(), cfg, reloj)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Cerrar()
	c, f, err := m.Configuracion()
	if err != nil || f == nil || len(c.EntradasCapacidad) != 2 || bytes.Equal(c.EntradasCapacidad[0].Material, c.EntradasCapacidad[1].Material) {
		t.Fatal("material no separado")
	}
	var b bytes.Buffer
	if err = m.EscribirMaterialPrivado(&b); err != nil {
		t.Fatal(err)
	}
	defer borrarBytes(b.Bytes())
	var p struct {
		Claves []struct {
			Audiencia string `json:"audiencia"`
			Secreto   []byte `json:"secreto_hmac"`
		} `json:"claves"`
	}
	if json.Unmarshal(b.Bytes(), &p) != nil || len(p.Claves) != 2 || len(p.Claves[0].Secreto) != 32 {
		t.Fatal("ABI privada incompatible")
	}
	for _, k := range p.Claves {
		borrarBytes(k.Secreto)
	}
	visible, err := json.Marshal(m)
	if err != nil || string(visible) != `{"material":"oculto"}` {
		t.Fatal("material exportado por JSON habitual")
	}
	m.Cerrar()
	if _, _, err = m.Configuracion(); err == nil {
		t.Fatal("material cerrado reutilizado")
	}
	if m.EscribirMaterialPrivado(&b) == nil {
		t.Fatal("material cerrado publicado")
	}
	if !bytes.Equal(c.EntradasCapacidad[0].Material, make([]byte, 32)) {
		t.Fatal("material retenido tras cierre")
	}
}
func TestGobiernoUsuariosRechazaRaizAjenaYAudienciasNoCerradas(t *testing.T) {
	for _, caso := range []string{"raiz", "audiencia", "duplicada", "dominio", "tercera", "caducada", "gobierno"} {
		t.Run(caso, func(t *testing.T) {
			cfg, r := configuracionGobiernoUsuariosPrueba(t)
			switch caso {
			case "raiz":
				cfg.Raiz.Publica[0] ^= 1
			case "audiencia":
				cfg.Entradas[1].Audiencia = "vec.admin.perfiles.v1"
			case "duplicada":
				cfg.Entradas[1].Audiencia = cfg.Entradas[0].Audiencia
			case "dominio":
				cfg.Entradas[1].Dominio = cfg.Entradas[0].Dominio
			case "tercera":
				cfg.Entradas = append(cfg.Entradas, cfg.Entradas[0])
			case "caducada":
				cfg.Entradas[0].ValidaHasta = r.Ahora()
			case "gobierno":
				cfg.Gobierno.HuellaSHA256 = ""
			}
			if m, err := PrepararMaterialUsuariosAdmin(context.Background(), cfg, r); err == nil {
				m.Cerrar()
				t.Fatal("fuente incompatible aceptada")
			}
		})
	}
}

func TestGobiernoUsuariosHMACIndependienteDelFirmanteFijado(t *testing.T) {
	cfg, r := configuracionGobiernoUsuariosPrueba(t)
	otra, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(nuevoDerivadorIdempotenciaPrueba(t, 3, 1), r.Ahora())
	if err != nil {
		t.Fatal(err)
	}
	defer borrarBytes(otra.privada)
	defer borrarBytes(otra.claveHMAC)
	cfg.Raiz.Publica = append(ed25519.PublicKey(nil), otra.privada.Public().(ed25519.PublicKey)...)
	if os.WriteFile(cfg.ArchivoSemillaRaiz, otra.privada.Seed(), 0600) != nil {
		t.Fatal("semilla sintetica")
	}
	raiz, err := confianza.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(cfg.Raiz.ClaveID, cfg.Raiz.Version, cfg.Raiz.Publica, cfg.Raiz.Audiencia, cfg.Raiz.Estado, cfg.Raiz.ValidaDesde, cfg.Raiz.ValidaHasta, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	gov, err := confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(cfg.Gobierno.Revision, cfg.Gobierno.Secuencia, cfg.Gobierno.PublicadaEn, cfg.Gobierno.ExpiraEn, raiz)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Gobierno.HuellaSHA256, err = gov.HuellaSHA256ParaGobierno()
	if err != nil {
		t.Fatal(err)
	}
	m, err := PrepararMaterialUsuariosAdmin(t.Context(), cfg, r)
	if err != nil {
		t.Fatal("firmante independiente fijado rechazado")
	}
	defer m.Cerrar()
	cfg.ArchivoSemillaRaiz = ""
	if x, err := PrepararMaterialUsuariosAdmin(t.Context(), cfg, r); err == nil {
		x.Cerrar()
		t.Fatal("firmante por defecto aceptado")
	}
}
