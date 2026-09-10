package bootstrap

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	core "vec-diputacion-granada/internal/vec/domain"
)

func TestIncorporacionV2CargaArchivoPrivado(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	c := archivoIncorporacionV2{Esquema: "vec.contratacion-temporal.incorporacion-servidor.v2",
		Referencias: ReferenciasCTIncorporacionDesarrollo{PrincipalV3Ref: "principal:prueba", PerfilV3Ref: "perfil:prueba", OrganizacionRef: "ref:" + strings.Repeat("a", 64), UnidadRef: "ref:" + strings.Repeat("b", 64), ActorRef: "ref:" + strings.Repeat("c", 64)}, Pools: map[string]string{}}
	c.MotivoAlta = core.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "motivo_11111111111111111111111111111111"}
	c.MotivoLectura = c.MotivoAlta
	for k := range rolesPoolsIncorporacionV2 {
		c.Pools[k] = k + ".dsn"
	}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(dir, "incorporacion.json")
	escribir := func(b []byte) {
		t.Helper()
		if err := os.WriteFile(ruta, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	escribir(b)
	x, raiz, err := leerConfiguracionIncorporacionV2(ruta)
	if err != nil || raiz == nil || x.Referencias != c.Referencias {
		t.Fatal("configuración privada nominal no recuperada")
	}
	raiz.Close()
	for nombre, datos := range map[string][]byte{
		"ruta_permisos_desconocida": append([]byte(`{"permiso":true,`), b[1:]...),
		"dos_documentos":            append(append([]byte(nil), b...), []byte(`{}`)...),
		"rol_no_declarado":          []byte(strings.Replace(string(b), `"raices_ct":`, `"admin":`, 1)),
		"referencia_ajena":          []byte(strings.Replace(string(b), c.Referencias.OrganizacionRef, "organizacion:con espacio", 1)),
	} {
		t.Run(nombre, func(t *testing.T) {
			escribir(datos)
			if _, r, e := leerConfiguracionIncorporacionV2(ruta); e == nil || r != nil {
				if r != nil {
					r.Close()
				}
				t.Fatal("configuración ambigua aceptada")
			}
		})
	}
	escribir(b)
	if err := os.Chmod(ruta, 0644); err != nil {
		t.Fatal(err)
	}
	if _, r, e := leerConfiguracionIncorporacionV2(ruta); e == nil || r != nil {
		t.Fatal("archivo no privado aceptado")
	}
}

func TestIncorporacionV2CargaMaterialYConfinamiento(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	r, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	semilla := make([]byte, 32)
	if _, err := rand.Read(semilla); err != nil {
		t.Fatal(err)
	}
	defer clear(semilla)
	privada := ed25519.NewKeyFromSeed(semilla)
	defer clear(privada)
	raizPublica, err := confianza.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA("clave:prueba:inc", 1, privada.Public().(ed25519.PublicKey), audienciaAtestacionContratacionTemporalDesarrollo, confianza.EstadoClaveAtestacionAutorizacionV3Activa, ahora.Add(-time.Hour), ahora.Add(time.Hour), time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "semilla"), semilla, 0600); err != nil {
		t.Fatal(err)
	}
	c := archivoMaterialIncorporacionV2{}
	for i, p := range []*archivoCapacidadIncorporacionV2{&c.Alta, &c.Lectura, &c.CT} {
		nombre := []string{"alta", "lectura", "ct"}[i]
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			t.Fatal(err)
		}
		h := sha256.Sum256(b)
		if err := os.WriteFile(filepath.Join(dir, nombre), b, 0600); err != nil {
			t.Fatal(err)
		}
		clear(b)
		*p = archivoCapacidadIncorporacionV2{ClaveID: "clave:prueba:" + nombre, Version: 1, File: nombre, SHA256: hex.EncodeToString(h[:]), EmisorID: "emisor:prueba:inc", Desde: ahora.Add(-time.Hour), Hasta: ahora.Add(time.Hour), RevisionGobierno: 1, HuellaGobierno: strings.Repeat("a", 64)}
	}
	e, err := cargarMaterialIncorporacionV2(r, c, raizPublica, relojContratacionTemporalDesarrollo{})
	if err != nil || e.Alta.Emisor == nil || e.Lectura.Emisor == nil || e.CT.Emisor == nil || e.Alta.Emisor == e.Lectura.Emisor {
		t.Fatal("material real no compuesto")
	}
	for nombre, mutar := range map[string]func(*archivoMaterialIncorporacionV2){
		"huella":             func(c *archivoMaterialIncorporacionV2) { c.Lectura.SHA256 = strings.Repeat("0", 64) },
		"fuera_de_raiz":      func(c *archivoMaterialIncorporacionV2) { c.Alta.File = "../semilla" },
		"clave_reutilizada":  func(c *archivoMaterialIncorporacionV2) { c.CT.File = c.Alta.File },
		"capacidad_caducada": func(c *archivoMaterialIncorporacionV2) { c.CT.Hasta = ahora.Add(-time.Second) },
	} {
		t.Run(nombre, func(t *testing.T) {
			x := c
			mutar(&x)
			if a, err := cargarMaterialIncorporacionV2(r, x, raizPublica, relojContratacionTemporalDesarrollo{}); err == nil || a.Alta.Emisor != nil {
				t.Fatal("material inválido aceptado")
			}
		})
	}
	if err := os.Symlink("semilla", filepath.Join(dir, "enlace")); err != nil {
		t.Fatal(err)
	}
	if b, err := leerArchivoIncorporacionV2(r, "enlace", 32); err == nil || b != nil {
		t.Fatal("symlink aceptado")
	}
	// Sólo constructores/archivos locales; no acredita gobierno instalado.
}

func TestIncorporacionV2SelectorArranqueYAmbitoNominal(t *testing.T) {
	t.Setenv(config.EnvIncorporacionV2File, "/privado/incorporacion.json")
	if config.Load().IncorporacionV2File != "/privado/incorporacion.json" {
		t.Fatal("CLI no carga selector")
	}
	for _, nuevo := range []func(config.Config) error{
		func(c config.Config) error { _, e := NewHTTPServerWithConfig(c); return e },
		func(c config.Config) error { _, e := NewHTTPServerPublicoWithConfig(c); return e },
	} {
		if err := nuevo(config.Config{IncorporacionV2File: "/privado/incorporacion.json"}); !errors.Is(err, ErrActivacionDesarrolloInvalida) {
			t.Fatal("selector activado fuera de superficie privada")
		}
	}
	alta, consulta, principal := escenarioConsultasRRHHDesarrolloPrueba(t)
	v, _ := alta.soporte.contexto.Vinculo.Datos()
	refs := ReferenciasCTIncorporacionDesarrollo{PrincipalV3Ref: v.PrincipalID, PerfilV3Ref: v.PerfilActivoRef, OrganizacionRef: "organizacion:desarrollo:dipgra", UnidadRef: "ref:" + strings.Repeat("b", 64), ActorRef: "ref:" + strings.Repeat("c", 64)}
	a := &contextoDetalleNominalIncorporacionV2{consulta, refs, alta.soporte.reloj}
	ctx := contextoRutaCoberturaDesarrolloPrueba(alta.soporte, principal, httpinterno.RutaIncorporacionEjercicioV2)
	c := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	c.certificadoVerificadoEn = alta.soporte.reloj.Ahora().Add(-time.Second)
	c.certificadoValidoHasta = alta.soporte.reloj.Ahora().Add(time.Minute)
	ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, c)
	hijo, err := contextoDetalleIncorporacionV2Desarrollo(ctx, alta.soporte)
	if err != nil {
		t.Fatal(err)
	}
	obtenido, err := a.ResolverContextoConsultaRRHH(hijo)
	if err != nil || obtenido.OrganizacionRef() != refs.OrganizacionRef {
		t.Fatal("consulta no ligada al ámbito real del plan")
	}
	if _, err := a.ResolverContextoConsultaRRHH(ctx); !errors.Is(err, ct.ErrAutorizacionDenegada) {
		t.Fatal("acepta petición sin sello")
	}
	a.referencias.PerfilV3Ref = "perfil:otro"
	if _, err := a.ResolverContextoConsultaRRHH(hijo); !errors.Is(err, ct.ErrAutorizacionDenegada) {
		t.Fatal("acepta perfil cruzado")
	}
}
