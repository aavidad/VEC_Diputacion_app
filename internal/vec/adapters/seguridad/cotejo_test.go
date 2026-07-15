package seguridad

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"testing"

	"vec-diputacion-granada/internal/vec/ports"
)

const (
	cotejoSeguridadPruebaSecreto           = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	cotejoSeguridadPruebaSecretoFormateado = "2345-6789 ABCD-EFGH JKLM-NPQR STUV-WXYZ"
)

func TestCotejoSeguridadHMACEsDeterministaSeparadoYAdmiteRotacion(t *testing.T) {
	configuracion := cotejoSeguridadPruebaConfiguracion()
	adaptador := cotejoSeguridadPruebaNuevoAdaptador(t, configuracion)
	secreto := cotejoSeguridadPruebaNuevoSecreto(t, cotejoSeguridadPruebaSecretoFormateado)
	ctx := context.Background()

	indiceA, err := adaptador.SellarIndiceCodigoCotejo(ctx, secreto)
	if err != nil {
		t.Fatalf("sellar primer indice: %v", err)
	}
	indiceB, err := adaptador.SellarIndiceCodigoCotejo(ctx, secreto)
	if err != nil {
		t.Fatalf("sellar segundo indice: %v", err)
	}
	esperadoActual := cotejoSeguridadPruebaHMAC(
		configuracion.ClaveIndiceActual.Identificador,
		configuracion.ClaveIndiceActual.Material,
		[]byte(cotejoSeguridadPruebaSecreto),
	)
	if indiceA != indiceB || indiceA != esperadoActual || strings.Contains(indiceA, cotejoSeguridadPruebaSecreto) {
		t.Fatalf("indices no deterministas o con fuga: %q / %q", indiceA, indiceB)
	}

	indices, err := adaptador.SellarIndicesConsultaCodigoCotejo(ctx, secreto)
	if err != nil {
		t.Fatalf("sellar indices de consulta: %v", err)
	}
	esperados := []string{esperadoActual}
	for _, historica := range configuracion.ClavesIndiceHistoricas {
		esperados = append(esperados, cotejoSeguridadPruebaHMAC(
			historica.Identificador, historica.Material, []byte(cotejoSeguridadPruebaSecreto),
		))
	}
	if !cotejoSeguridadPruebaCadenasIguales(indices, esperados) {
		t.Fatalf("rotacion = %v; esperada %v", indices, esperados)
	}

	datosSolicitud := []byte(cotejoSeguridadPruebaSecreto)
	solicitudA, err := adaptador.SellarSolicitudCotejo(ctx, datosSolicitud)
	if err != nil {
		t.Fatalf("sellar solicitud: %v", err)
	}
	solicitudB, _ := adaptador.SellarSolicitudCotejo(ctx, datosSolicitud)
	esperadaSolicitud := cotejoSeguridadPruebaHMAC(
		configuracion.ClaveSolicitud.Identificador, configuracion.ClaveSolicitud.Material, datosSolicitud,
	)
	if solicitudA != solicitudB || solicitudA != esperadaSolicitud || solicitudA == indiceA {
		t.Fatalf("separacion por finalidad incorrecta: indice=%q solicitud=%q", indiceA, solicitudA)
	}

	otroSecreto := cotejoSeguridadPruebaNuevoSecreto(t, "ZZZZZZZZZZZZZZZZZZZZZZZZZZ")
	otroIndice, err := adaptador.SellarIndiceCodigoCotejo(ctx, otroSecreto)
	if err != nil || otroIndice == indiceA {
		t.Fatalf("otro secreto produjo el mismo indice: %q, %v", otroIndice, err)
	}
}

func TestCotejoSeguridadConstructorCopiaClavesYRechazaConfiguracionInsegura(t *testing.T) {
	t.Run("copias defensivas", func(t *testing.T) {
		configuracion := cotejoSeguridadPruebaConfiguracion()
		idActual := configuracion.ClaveIndiceActual.Identificador
		claveActual := append([]byte(nil), configuracion.ClaveIndiceActual.Material...)
		idHistorica := configuracion.ClavesIndiceHistoricas[0].Identificador
		claveHistorica := append([]byte(nil), configuracion.ClavesIndiceHistoricas[0].Material...)
		idSolicitud := configuracion.ClaveSolicitud.Identificador
		claveSolicitud := append([]byte(nil), configuracion.ClaveSolicitud.Material...)
		adaptador := cotejoSeguridadPruebaNuevoAdaptador(t, configuracion)

		cotejoSeguridadPruebaRellenar(configuracion.ClaveIndiceActual.Material, 0)
		cotejoSeguridadPruebaRellenar(configuracion.ClavesIndiceHistoricas[0].Material, 0)
		cotejoSeguridadPruebaRellenar(configuracion.ClaveSolicitud.Material, 0)
		configuracion.ClaveIndiceActual.Identificador = "indice_alterado"
		configuracion.ClavesIndiceHistoricas[0].Identificador = "historica_alterada"
		configuracion.ClaveSolicitud.Identificador = "solicitud_alterada"

		secreto := cotejoSeguridadPruebaNuevoSecreto(t, cotejoSeguridadPruebaSecreto)
		indice, err := adaptador.SellarIndiceCodigoCotejo(context.Background(), secreto)
		if err != nil || indice != cotejoSeguridadPruebaHMAC(idActual, claveActual, []byte(cotejoSeguridadPruebaSecreto)) {
			t.Fatalf("la mutacion externa altero el indice: %q, %v", indice, err)
		}
		indices, err := adaptador.SellarIndicesConsultaCodigoCotejo(context.Background(), secreto)
		if err != nil || indices[1] != cotejoSeguridadPruebaHMAC(idHistorica, claveHistorica, []byte(cotejoSeguridadPruebaSecreto)) {
			t.Fatalf("la mutacion externa altero la historia: %v, %v", indices, err)
		}
		solicitud, err := adaptador.SellarSolicitudCotejo(context.Background(), []byte("solicitud estable"))
		if err != nil || solicitud != cotejoSeguridadPruebaHMAC(idSolicitud, claveSolicitud, []byte("solicitud estable")) {
			t.Fatalf("la mutacion externa altero la solicitud: %q, %v", solicitud, err)
		}
	})

	casos := []struct {
		nombre   string
		preparar func(*ConfiguracionCriptografiaCotejo)
	}{
		{"version vacia", func(c *ConfiguracionCriptografiaCotejo) { c.VersionGenerador = "" }},
		{"version no canonica", func(c *ConfiguracionCriptografiaCotejo) { c.VersionGenerador = " generador-v1" }},
		{"identificador invalido", func(c *ConfiguracionCriptografiaCotejo) { c.ClaveIndiceActual.Identificador = "Indice Actual" }},
		{"clave actual corta", func(c *ConfiguracionCriptografiaCotejo) { c.ClaveIndiceActual.Material = []byte("corta") }},
		{"clave solicitud corta", func(c *ConfiguracionCriptografiaCotejo) { c.ClaveSolicitud.Material = nil }},
		{"id reutilizado por finalidad", func(c *ConfiguracionCriptografiaCotejo) {
			c.ClaveSolicitud.Identificador = c.ClaveIndiceActual.Identificador
		}},
		{"material reutilizado por finalidad", func(c *ConfiguracionCriptografiaCotejo) {
			c.ClaveSolicitud.Material = append([]byte(nil), c.ClaveIndiceActual.Material...)
		}},
		{"id historico duplicado", func(c *ConfiguracionCriptografiaCotejo) {
			c.ClavesIndiceHistoricas[1].Identificador = c.ClavesIndiceHistoricas[0].Identificador
		}},
		{"material historico duplicado", func(c *ConfiguracionCriptografiaCotejo) {
			c.ClavesIndiceHistoricas[1].Material = append([]byte(nil), c.ClavesIndiceHistoricas[0].Material...)
		}},
		{"demasiadas claves historicas", func(c *ConfiguracionCriptografiaCotejo) {
			c.ClavesIndiceHistoricas = make([]ConfiguracionClaveHMACCotejo, maximoClavesHistoricasCotejo+1)
			for indice := range c.ClavesIndiceHistoricas {
				c.ClavesIndiceHistoricas[indice] = ConfiguracionClaveHMACCotejo{
					Identificador: fmt.Sprintf("indice_historico_%02d", indice),
					Material:      []byte(strings.Repeat(string(rune('a'+indice%20)), longitudMinimaClaveHMACCotejo)),
				}
			}
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			configuracion := cotejoSeguridadPruebaConfiguracion()
			caso.preparar(&configuracion)
			adaptador, err := NuevoAdaptadorCriptograficoCotejo(configuracion)
			if adaptador != nil || !errors.Is(err, ErrConfiguracionCriptografiaCotejoInvalida) {
				t.Fatalf("constructor = (%v, %v)", adaptador, err)
			}
		})
	}
}

func TestCotejoSeguridadGeneraCSVConEntropiaEIdentificadoresOpacos(t *testing.T) {
	adaptador := cotejoSeguridadPruebaNuevoAdaptador(t, cotejoSeguridadPruebaConfiguracion())
	valores := make(map[string]struct{})
	identificadores := make(map[string]struct{})
	patronID := regexp.MustCompile(`^codigo-cotejo-[0-9a-f]{40}$`)
	for intento := 0; intento < 256; intento++ {
		generado, err := adaptador.GenerarValorCodigoCotejo(context.Background())
		if err != nil {
			t.Fatalf("generar CSV %d: %v", intento, err)
		}
		valor := generado.Secreto.Revelar()
		if generado.EntropiaBits != 130 || generado.EntropiaBits < 128 ||
			generado.VersionGenerador != "generador-cotejo-v1" || len(valor) != longitudValorCodigoCotejo ||
			generado.Secreto.Validar() != nil {
			t.Fatalf("CSV generado invalido: %+v", generado)
		}
		for _, caracter := range valor {
			if !strings.ContainsRune(alfabetoCodigoCotejo, caracter) {
				t.Fatalf("caracter no permitido %q en %q", caracter, valor)
			}
		}
		if _, repetido := valores[valor]; repetido {
			t.Fatalf("CSV repetido: %q", valor)
		}
		valores[valor] = struct{}{}
		texto := fmt.Sprintf("%v|%+v|%#v", generado.Secreto, generado.Secreto, generado.Secreto)
		if strings.Contains(texto, valor) {
			t.Fatalf("el formato revela el CSV: %s", texto)
		}
		if _, err := json.Marshal(generado.Secreto); !errors.Is(err, ports.ErrSerializacionCodigoCotejoProhibida) {
			t.Fatalf("serializar CSV: error = %v", err)
		}

		identificador, err := adaptador.NuevoIDCodigoCotejo()
		if err != nil || !patronID.MatchString(identificador) {
			t.Fatalf("identificador opaco = %q, %v", identificador, err)
		}
		if _, repetido := identificadores[identificador]; repetido {
			t.Fatalf("identificador repetido: %q", identificador)
		}
		identificadores[identificador] = struct{}{}
	}
}

func TestCotejoSeguridadFallaCerradoYBorraSusCopias(t *testing.T) {
	adaptador := cotejoSeguridadPruebaNuevoAdaptador(t, cotejoSeguridadPruebaConfiguracion())
	secreto := cotejoSeguridadPruebaNuevoSecreto(t, cotejoSeguridadPruebaSecreto)
	ctxCancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := adaptador.GenerarValorCodigoCotejo(ctxCancelado); !errors.Is(err, context.Canceled) {
		t.Fatalf("generador con contexto cancelado: %v", err)
	}
	if _, err := adaptador.SellarIndiceCodigoCotejo(ctxCancelado, secreto); !errors.Is(err, context.Canceled) {
		t.Fatalf("indice con contexto cancelado: %v", err)
	}
	if _, err := adaptador.SellarIndicesConsultaCodigoCotejo(ctxCancelado, secreto); !errors.Is(err, context.Canceled) {
		t.Fatalf("consulta con contexto cancelado: %v", err)
	}
	if _, err := adaptador.SellarSolicitudCotejo(ctxCancelado, []byte("datos")); !errors.Is(err, context.Canceled) {
		t.Fatalf("solicitud con contexto cancelado: %v", err)
	}
	if _, err := adaptador.SellarIndiceCodigoCotejo(context.Background(), ports.SecretoCodigoCotejo{}); !errors.Is(err, ErrMaterialCriptograficoCotejoInvalido) {
		t.Fatalf("secreto cero aceptado: %v", err)
	}
	if _, err := adaptador.SellarSolicitudCotejo(context.Background(), nil); !errors.Is(err, ErrMaterialCriptograficoCotejoInvalido) {
		t.Fatalf("solicitud vacia aceptada: %v", err)
	}
	if _, err := adaptador.SellarIndiceCodigoCotejo(nil, secreto); !errors.Is(err, ErrMaterialCriptograficoCotejoInvalido) {
		t.Fatalf("contexto nulo aceptado: %v", err)
	}

	actual := adaptador.claveIndiceActual.material
	historica := adaptador.clavesIndiceHistoricas[0].material
	solicitud := adaptador.claveSolicitud.material
	adaptador.Cerrar()
	adaptador.Cerrar()
	for nombre, material := range map[string][]byte{"actual": actual, "historica": historica, "solicitud": solicitud} {
		for _, valor := range material {
			if valor != 0 {
				t.Fatalf("la copia %s no fue borrada: %v", nombre, material)
			}
		}
	}
	if _, err := adaptador.GenerarValorCodigoCotejo(context.Background()); !errors.Is(err, ErrCriptografiaCotejoCerrada) {
		t.Fatalf("generar tras cerrar: %v", err)
	}
	if _, err := adaptador.NuevoIDCodigoCotejo(); !errors.Is(err, ErrCriptografiaCotejoCerrada) {
		t.Fatalf("ID tras cerrar: %v", err)
	}
	if _, err := adaptador.SellarIndiceCodigoCotejo(context.Background(), secreto); !errors.Is(err, ErrCriptografiaCotejoCerrada) {
		t.Fatalf("indice tras cerrar: %v", err)
	}
	if _, err := adaptador.SellarSolicitudCotejo(context.Background(), []byte("datos")); !errors.Is(err, ErrCriptografiaCotejoCerrada) {
		t.Fatalf("solicitud tras cerrar: %v", err)
	}
	var nulo *AdaptadorCriptograficoCotejo
	if _, err := nulo.NuevoIDCodigoCotejo(); !errors.Is(err, ErrConfiguracionCriptografiaCotejoInvalida) {
		t.Fatalf("adaptador nulo: %v", err)
	}
	if _, err := nulo.SellarIndiceCodigoCotejo(context.Background(), secreto); !errors.Is(err, ErrConfiguracionCriptografiaCotejoInvalida) {
		t.Fatalf("indice con adaptador nulo: %v", err)
	}
	if _, err := nulo.SellarIndicesConsultaCodigoCotejo(context.Background(), secreto); !errors.Is(err, ErrConfiguracionCriptografiaCotejoInvalida) {
		t.Fatalf("consulta con adaptador nulo: %v", err)
	}
	if _, err := nulo.SellarSolicitudCotejo(context.Background(), []byte("datos")); !errors.Is(err, ErrConfiguracionCriptografiaCotejoInvalida) {
		t.Fatalf("solicitud con adaptador nulo: %v", err)
	}
}

func TestCotejoSeguridadNoFiltraClavesEnTextoJSONOFormato(t *testing.T) {
	configuracion := cotejoSeguridadPruebaConfiguracion()
	adaptador := cotejoSeguridadPruebaNuevoAdaptador(t, configuracion)
	claveDirecta := configuracion.ClaveIndiceActual
	materiales := [][]byte{
		configuracion.ClaveIndiceActual.Material,
		configuracion.ClavesIndiceHistoricas[0].Material,
		configuracion.ClavesIndiceHistoricas[1].Material,
		configuracion.ClaveSolicitud.Material,
	}
	fragmentos := []string{
		fmt.Sprintf("%v|%+v|%#v|%s", claveDirecta, claveDirecta, claveDirecta, claveDirecta),
		fmt.Sprintf("%v|%+v|%#v|%s", configuracion, configuracion, configuracion, configuracion),
		fmt.Sprintf("%v|%+v|%#v|%s", adaptador, adaptador, adaptador, adaptador),
	}
	for _, valor := range []any{claveDirecta, configuracion, adaptador} {
		contenido, err := json.Marshal(valor)
		if err != nil {
			t.Fatalf("serializar representacion protegida: %v", err)
		}
		if !json.Valid(contenido) {
			t.Fatalf("JSON invalido: %s", contenido)
		}
		fragmentos = append(fragmentos, string(contenido))
	}
	var registro bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&registro, nil))
	logger.Info("comprobar redaccion",
		slog.Any("clave", claveDirecta),
		slog.Any("configuracion", configuracion),
		slog.Any("adaptador", adaptador),
	)
	fragmentos = append(fragmentos, registro.String())
	texto := strings.Join(fragmentos, "\n")
	for _, material := range materiales {
		for _, prohibido := range []string{string(material), fmt.Sprint(material)} {
			if strings.Contains(texto, prohibido) {
				t.Fatalf("representacion filtra material de clave %q: %s", prohibido, texto)
			}
		}
	}
}

func TestCotejoSeguridadEsSeguroEnConcurrencia(t *testing.T) {
	adaptador := cotejoSeguridadPruebaNuevoAdaptador(t, cotejoSeguridadPruebaConfiguracion())
	secreto := cotejoSeguridadPruebaNuevoSecreto(t, cotejoSeguridadPruebaSecreto)
	indiceEsperado, err := adaptador.SellarIndiceCodigoCotejo(context.Background(), secreto)
	if err != nil {
		t.Fatalf("preparar indice esperado: %v", err)
	}
	solicitudEsperada, err := adaptador.SellarSolicitudCotejo(context.Background(), []byte("solicitud concurrente"))
	if err != nil {
		t.Fatalf("preparar solicitud esperada: %v", err)
	}

	const trabajadores = 96
	type resultado struct {
		indice        string
		solicitud     string
		valor         string
		identificador string
		err           error
	}
	resultados := make(chan resultado, trabajadores)
	var grupo sync.WaitGroup
	grupo.Add(trabajadores)
	for contador := 0; contador < trabajadores; contador++ {
		go func() {
			defer grupo.Done()
			indice, err := adaptador.SellarIndiceCodigoCotejo(context.Background(), secreto)
			if err != nil {
				resultados <- resultado{err: err}
				return
			}
			indices, err := adaptador.SellarIndicesConsultaCodigoCotejo(context.Background(), secreto)
			if err != nil {
				resultados <- resultado{err: err}
				return
			}
			if len(indices) != 3 || indices[0] != indice {
				resultados <- resultado{err: fmt.Errorf("rotacion concurrente invalida: %v", indices)}
				return
			}
			solicitud, err := adaptador.SellarSolicitudCotejo(context.Background(), []byte("solicitud concurrente"))
			if err != nil {
				resultados <- resultado{err: err}
				return
			}
			generado, err := adaptador.GenerarValorCodigoCotejo(context.Background())
			if err != nil {
				resultados <- resultado{err: err}
				return
			}
			identificador, err := adaptador.NuevoIDCodigoCotejo()
			resultados <- resultado{
				indice: indice, solicitud: solicitud, valor: generado.Secreto.Revelar(),
				identificador: identificador, err: err,
			}
		}()
	}
	grupo.Wait()
	close(resultados)
	valores := make(map[string]struct{}, trabajadores)
	identificadores := make(map[string]struct{}, trabajadores)
	for resultado := range resultados {
		if resultado.err != nil {
			t.Fatalf("operacion concurrente: %v", resultado.err)
		}
		if resultado.indice != indiceEsperado || resultado.solicitud != solicitudEsperada {
			t.Fatalf("HMAC concurrente no determinista: %+v", resultado)
		}
		if _, existe := valores[resultado.valor]; existe {
			t.Fatalf("CSV concurrente repetido: %q", resultado.valor)
		}
		valores[resultado.valor] = struct{}{}
		if _, existe := identificadores[resultado.identificador]; existe {
			t.Fatalf("ID concurrente repetido: %q", resultado.identificador)
		}
		identificadores[resultado.identificador] = struct{}{}
	}
}

func cotejoSeguridadPruebaConfiguracion() ConfiguracionCriptografiaCotejo {
	return ConfiguracionCriptografiaCotejo{
		VersionGenerador: "generador-cotejo-v1",
		ClaveIndiceActual: ConfiguracionClaveHMACCotejo{
			Identificador: "indice_cotejo_2026_03", Material: []byte(strings.Repeat("A", 32)),
		},
		ClavesIndiceHistoricas: []ConfiguracionClaveHMACCotejo{
			{Identificador: "indice_cotejo_2026_02", Material: []byte(strings.Repeat("B", 32))},
			{Identificador: "indice_cotejo_2026_01", Material: []byte(strings.Repeat("C", 32))},
		},
		ClaveSolicitud: ConfiguracionClaveHMACCotejo{
			Identificador: "solicitud_cotejo_2026_01", Material: []byte(strings.Repeat("D", 32)),
		},
	}
}

func cotejoSeguridadPruebaNuevoAdaptador(
	t *testing.T,
	configuracion ConfiguracionCriptografiaCotejo,
) *AdaptadorCriptograficoCotejo {
	t.Helper()
	adaptador, err := NuevoAdaptadorCriptograficoCotejo(configuracion)
	if err != nil {
		t.Fatalf("crear adaptador criptografico: %v", err)
	}
	return adaptador
}

func cotejoSeguridadPruebaNuevoSecreto(t *testing.T, valor string) ports.SecretoCodigoCotejo {
	t.Helper()
	secreto, err := ports.NuevoSecretoCodigoCotejo(valor)
	if err != nil {
		t.Fatalf("crear secreto de cotejo: %v", err)
	}
	return secreto
}

func cotejoSeguridadPruebaHMAC(identificador string, clave, datos []byte) string {
	mac := hmac.New(sha256.New, clave)
	_, _ = mac.Write(datos)
	return "hmac-sha256:" + identificador + ":" + hex.EncodeToString(mac.Sum(nil))
}

func cotejoSeguridadPruebaCadenasIguales(primera, segunda []string) bool {
	if len(primera) != len(segunda) {
		return false
	}
	for indice := range primera {
		if primera[indice] != segunda[indice] {
			return false
		}
	}
	return true
}

func cotejoSeguridadPruebaRellenar(datos []byte, valor byte) {
	for indice := range datos {
		datos[indice] = valor
	}
}
