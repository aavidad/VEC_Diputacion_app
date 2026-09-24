package httpseguridad

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type acreditadorPasarelaPrueba struct{ falla bool }

func (a *acreditadorPasarelaPrueba) AcreditarIdentidadYFactores(_ context.Context, identidad AsercionProxyIdentidad) error {
	if a.falla || len(identidad.Factores) != 2 {
		return ErrAsercionPasarela
	}
	return nil
}

type revocacionPasarelaPrueba struct {
	emisorRevocado, claveRevocada, asercionRevocada, sesionRevocada, noDisponible bool
	ultima                                                                        ConsultaRevocacionPasarela
}

func (r *revocacionPasarelaPrueba) ComprobarEmisorActivo(_ context.Context, emisor string) error {
	if emisor == "" || r.emisorRevocado || r.noDisponible {
		return ErrAsercionPasarela
	}
	return nil
}

func (r *revocacionPasarelaPrueba) ComprobarClaveActiva(_ context.Context, emisor, claveID string) error {
	if emisor == "" || claveID == "" || r.claveRevocada || r.noDisponible {
		return ErrAsercionPasarela
	}
	return nil
}

func (r *revocacionPasarelaPrueba) ComprobarIdentidadActiva(_ context.Context, consulta ConsultaRevocacionPasarela) error {
	r.ultima = consulta
	if r.asercionRevocada || r.sesionRevocada || r.noDisponible {
		return ErrAsercionPasarela
	}
	return nil
}

type fuenteGarantiaPasarelaPrueba struct{ acreditada bool }

func (f *fuenteGarantiaPasarelaPrueba) AcreditarGarantiaAlta(_ context.Context, entrada EntradaEvaluacionGarantia) (DictamenGarantiaPasarela, error) {
	if !f.acreditada {
		return DictamenGarantiaPasarela{}, ErrAsercionNoValida
	}
	return DictamenGarantiaPasarela{Garantia: dominiovec.AuthAssuranceHigh,
		PoliticaRef: "pga_0123456789abcdefghijkl", HuellaPoliticaSHA256: strings.Repeat("a", 64),
		FactoresAcreditados: append([]FactorAutenticacion(nil), entrada.Factores...)}, nil
}

type entornoAsercionPasarela struct {
	configuracion ConfiguracionSuperficie
	servicio      *ServicioIdentidad
	canal         CanalProxyAutenticado
	emisor        *EmisorAsercionPasarela
	verificador   *VerificadorAsercionPasarela
	revocacion    *revocacionPasarelaPrueba
	acreditador   *acreditadorPasarelaPrueba
	reloj         *relojFijo
	claveID       string
	identidad     AsercionProxyIdentidad
}

func nuevoEntornoAsercionPasarela(t *testing.T) entornoAsercionPasarela {
	t.Helper()
	ahora := time.Date(2026, 9, 24, 20, 0, 0, 0, time.UTC)
	c := configuracionInternaValida()
	publica, privada, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	revocacion := &revocacionPasarelaPrueba{}
	acreditador := &acreditadorPasarelaPrueba{}
	reloj := &relojFijo{ahora: ahora}
	claveID := "clave_sintetica_01"
	emisor, err := NuevoEmisorAsercionPasarela(c, claveID, privada, acreditador, revocacion, reloj)
	if err != nil {
		t.Fatal(err)
	}
	verificador, err := NuevoVerificadorAsercionPasarela(c, map[string]ed25519.PublicKey{claveID: publica}, revocacion, reloj)
	if err != nil {
		t.Fatal(err)
	}
	evaluador, err := NuevoEvaluadorGarantiaPasarela(
		&fuenteGarantiaPasarelaPrueba{acreditada: true},
		"pga_0123456789abcdefghijkl", strings.Repeat("a", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	servicio := debeServicio(t, c, verificador, evaluador, nuevoRegistroMemoria(), reloj)
	canal := debeCanalTLS(t, servicio, c)
	return entornoAsercionPasarela{configuracion: c, servicio: servicio, canal: canal, emisor: emisor,
		verificador: verificador, revocacion: revocacion, acreditador: acreditador, reloj: reloj,
		claveID: claveID, identidad: asercionInternaValida(ahora, c, canal)}
}

func prepararPeticionPrueba(t *testing.T, metodo, ruta, cuerpo string) *http.Request {
	t.Helper()
	r := httptest.NewRequest(metodo, ruta, strings.NewReader(cuerpo))
	preparada, err := PrepararPeticionAsercionPasarela(r, limiteCuerpoPasarela)
	if err != nil {
		t.Fatalf("preparar peticion: %v", err)
	}
	return preparada
}

func (e entornoAsercionPasarela) emitirPara(t *testing.T, r *http.Request) []byte {
	t.Helper()
	vinculo, ok := r.Context().Value(claveContextoPeticionPasarela{}).(VinculoPeticionPasarela)
	if !ok {
		t.Fatal("no se fijo el vinculo privado")
	}
	protegida, err := e.emisor.Emitir(r.Context(), e.identidad, vinculo)
	if err != nil {
		t.Fatalf("emitir asercion: %v", err)
	}
	return protegida
}

func TestAsercionPasarelaFirmaVinculoYConsumoDurable(t *testing.T) {
	e := nuevoEntornoAsercionPasarela(t)
	r := prepararPeticionPrueba(t, "POST", "/vec-interno/seguimiento?exp=1", `{"accion":"consultar"}`)
	protegida := e.emitirPara(t, r)
	r.Header.Set(CabeceraAsercionPasarela, base64.RawURLEncoding.EncodeToString(protegida))
	extraida, err := (ExtractorAsercionPasarela{}).ExtraerAsercionProtegida(r)
	if err != nil || string(extraida) != string(protegida) {
		t.Fatalf("extraccion protegida: %v", err)
	}
	credencial := debeCredencial(t, extraida, e.canal)
	identidad, err := e.servicio.Resolver(r.Context(), credencial)
	if err != nil {
		t.Fatalf("resolver asercion firmada: %v", err)
	}
	cuenta, auditoria, err := e.servicio.ProyectarCuentaAutenticada(r.Context(), identidad)
	if err != nil || cuenta.Garantia != dominiovec.AuthAssuranceHigh ||
		auditoria.CanalVinculadoRef() != e.canal.ReferenciaVinculacion() {
		t.Fatalf("sesion no vinculada y revalidada: %v", err)
	}
	if e.revocacion.ultima.ClaveID != e.claveID || e.revocacion.ultima.AsercionID == "" ||
		len(e.revocacion.ultima.Factores) != 2 {
		t.Fatal("no se consulto la revocacion de clave, asercion y factores")
	}
	if _, err := e.servicio.Resolver(r.Context(), credencial); !errors.Is(err, ErrSesionNoValida) {
		t.Fatalf("replay no rechazado por el registro durable: %v", err)
	}
}

func TestAsercionPasarelaRechazaAlteracionesPeticionFirmaYCanal(t *testing.T) {
	e := nuevoEntornoAsercionPasarela(t)
	original := prepararPeticionPrueba(t, "POST", "/vec-interno/seguimiento?x=1", "cuerpo")
	protegida := e.emitirPara(t, original)
	for nombre, r := range map[string]*http.Request{
		"metodo": prepararPeticionPrueba(t, "GET", "/vec-interno/seguimiento?x=1", "cuerpo"),
		"ruta":   prepararPeticionPrueba(t, "POST", "/vec-interno/otro?x=1", "cuerpo"),
		"query":  prepararPeticionPrueba(t, "POST", "/vec-interno/seguimiento?x=2", "cuerpo"),
		"cuerpo": prepararPeticionPrueba(t, "POST", "/vec-interno/seguimiento?x=1", "distinto"),
	} {
		t.Run(nombre, func(t *testing.T) {
			if _, err := e.verificador.Verificar(r.Context(), protegida); !errors.Is(err, ErrAsercionPasarela) {
				t.Fatalf("vinculo ajeno aceptado: %v", err)
			}
		})
	}
	alterada := append([]byte(nil), protegida...)
	alterada[len(alterada)-4] ^= 1
	if _, err := e.verificador.Verificar(original.Context(), alterada); !errors.Is(err, ErrAsercionPasarela) {
		t.Fatalf("firma alterada aceptada: %v", err)
	}
	if _, err := e.verificador.Verificar(context.Background(), protegida); !errors.Is(err, ErrAsercionPasarela) {
		t.Fatalf("peticion ausente aceptada: %v", err)
	}
	// Una firma valida que afirma otro canal tampoco alcanza la sesion.
	e.identidad.CanalVinculadoRef = "tls-exportador:sha256:" + strings.Repeat("a", 64)
	otra := e.emitirPara(t, original)
	if _, err := e.servicio.Resolver(original.Context(), debeCredencial(t, otra, e.canal)); !errors.Is(err, ErrAsercionNoValida) {
		t.Fatalf("canal diferente aceptado: %v", err)
	}
}

func TestAsercionPasarelaDeniegaRevocacionYFuenteCaida(t *testing.T) {
	e := nuevoEntornoAsercionPasarela(t)
	r := prepararPeticionPrueba(t, "GET", "/vec-interno/seguimiento", "")
	protegida := e.emitirPara(t, r)
	for nombre, activar := range map[string]func(){
		"emisor":      func() { e.revocacion.emisorRevocado = true },
		"clave":       func() { e.revocacion.claveRevocada = true },
		"asercion":    func() { e.revocacion.asercionRevocada = true },
		"sesion":      func() { e.revocacion.sesionRevocada = true },
		"dependencia": func() { e.revocacion.noDisponible = true },
	} {
		t.Run(nombre, func(t *testing.T) {
			*e.revocacion = revocacionPasarelaPrueba{}
			activar()
			if _, err := e.verificador.Verificar(r.Context(), protegida); !errors.Is(err, ErrAsercionPasarela) {
				t.Fatalf("revocacion o caida aceptada: %v", err)
			}
		})
	}
	*e.revocacion = revocacionPasarelaPrueba{}
	e.acreditador.falla = true
	vinculo := r.Context().Value(claveContextoPeticionPasarela{}).(VinculoPeticionPasarela)
	if _, err := e.emisor.Emitir(r.Context(), e.identidad, vinculo); !errors.Is(err, ErrAsercionPasarela) {
		t.Fatalf("factor no acreditado se firmo: %v", err)
	}
}

func TestAsercionPasarelaDeniegaAudienciaDistintaYEntradasLibres(t *testing.T) {
	e := nuevoEntornoAsercionPasarela(t)
	r := prepararPeticionPrueba(t, "GET", "/vec-interno/seguimiento", "")
	e.identidad.Audiencia = "otra-audiencia"
	vinculo := r.Context().Value(claveContextoPeticionPasarela{}).(VinculoPeticionPasarela)
	if _, err := e.emisor.Emitir(r.Context(), e.identidad, vinculo); !errors.Is(err, ErrAsercionPasarela) {
		t.Fatalf("audiencia distinta firmada: %v", err)
	}
	e.identidad.Audiencia = e.configuracion.Audiencia
	protegida := e.emitirPara(t, r)
	otraAudiencia := e.configuracion
	otraAudiencia.Audiencia = "vec-interna-otra"
	verificadorAjeno, err := NuevoVerificadorAsercionPasarela(
		otraAudiencia, e.verificador.claves, e.revocacion, e.reloj,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verificadorAjeno.Verificar(r.Context(), protegida); !errors.Is(err, ErrAsercionPasarela) {
		t.Fatalf("audiencia ajena verificada: %v", err)
	}
	r.Header.Set(CabeceraAsercionPasarela, base64.RawURLEncoding.EncodeToString(protegida))
	for nombre, cabecera := range map[string]string{
		"bearer": "Authorization", "cookie": "Cookie", "identidad libre": "X-Forwarded-User",
		"certificado reenviado": "X-SSL-Client-Cert",
	} {
		t.Run(nombre, func(t *testing.T) {
			copia := r.Clone(r.Context())
			copia.Header.Set(cabecera, "valor-sintetico")
			if _, err := (ExtractorAsercionPasarela{}).ExtraerAsercionProtegida(copia); !errors.Is(err, ErrAsercionPasarela) {
				t.Fatalf("entrada libre aceptada: %v", err)
			}
		})
	}
	r.Header.Add(CabeceraAsercionPasarela, "duplicada")
	if _, err := (ExtractorAsercionPasarela{}).ExtraerAsercionProtegida(r); !errors.Is(err, ErrAsercionPasarela) {
		t.Fatalf("cabecera duplicada aceptada: %v", err)
	}
	if _, err := PrepararPeticionAsercionPasarela(httptest.NewRequest("GET", "/a/%2f/b", nil), 10); !errors.Is(err, ErrAsercionPasarela) {
		t.Fatalf("ruta ambigua aceptada: %v", err)
	}
	if _, err := PrepararPeticionAsercionPasarela(httptest.NewRequest("POST", "/a", strings.NewReader("12")), 1); !errors.Is(err, ErrAsercionPasarela) {
		t.Fatalf("cuerpo mayor que limite aceptado: %v", err)
	}
}

func TestEvaluadorPasarelaExigeFuenteAltaAcreditada(t *testing.T) {
	if _, err := NuevoEvaluadorGarantiaPasarela(nil, "pga_0123456789abcdefghijkl", strings.Repeat("a", 64)); !errors.Is(err, ErrEvaluadorGarantiaAusente) {
		t.Fatalf("evaluador sin fuente: %v", err)
	}
	e := nuevoEntornoAsercionPasarela(t)
	fuente := &fuenteGarantiaPasarelaPrueba{}
	evaluador, err := NuevoEvaluadorGarantiaPasarela(fuente, "pga_0123456789abcdefghijkl", strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	entrada := EntradaEvaluacionGarantia{ACRVerificado: e.identidad.ACRVerificado,
		Emisor: e.identidad.Emisor, Superficie: e.identidad.Superficie,
		SujetoID: e.identidad.SujetoID, CuentaID: e.identidad.Cuenta.ID,
		MetodoPrimario: e.identidad.MetodoPrimario, Factores: e.identidad.Factores}
	if _, err := evaluador.Evaluar(context.Background(), entrada); !errors.Is(err, ErrAsercionNoValida) {
		t.Fatalf("ACR libre produjo garantia alta: %v", err)
	}
	fuente.acreditada = true
	resultado, err := evaluador.Evaluar(context.Background(), entrada)
	if err != nil || resultado.Garantia != dominiovec.AuthAssuranceHigh {
		t.Fatalf("fuente acreditada no produjo garantia: %v", err)
	}
	entrada.Factores[1].GrupoCriptograficoRef = entrada.Factores[0].GrupoCriptograficoRef
	if _, err := evaluador.Evaluar(context.Background(), entrada); !errors.Is(err, ErrAsercionNoValida) {
		t.Fatalf("factores dependientes aceptados: %v", err)
	}
}
