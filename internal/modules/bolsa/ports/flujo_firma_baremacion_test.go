package ports

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestTokenArrendamientoFlujoFirmaEsOpacoYFallaCerrado(t *testing.T) {
	token, err := NuevoTokenArrendamientoFlujoFirmaBaremacion()
	if err != nil || token.Validar() != nil {
		t.Fatalf("NuevoTokenArrendamientoFlujoFirmaBaremacion() = (%v, %v)", token, err)
	}
	ajeno, err := NuevoTokenArrendamientoFlujoFirmaBaremacion()
	if err != nil {
		t.Fatal(err)
	}
	clave := bytes.Repeat([]byte{0x31}, 32)
	huella, err := token.HuellaHMACSHA256(clave)
	if err != nil || len(huella) != 32 || !token.CoincideHuellaHMACSHA256(clave, huella) {
		t.Fatalf("huella valida no reconocida: longitud=%d error=%v", len(huella), err)
	}
	alterada := append([]byte(nil), huella...)
	alterada[len(alterada)-1] ^= 0xff
	if token.CoincideHuellaHMACSHA256(clave, alterada) ||
		token.CoincideHuellaHMACSHA256(bytes.Repeat([]byte{0x32}, 32), huella) ||
		ajeno.CoincideHuellaHMACSHA256(clave, huella) ||
		(TokenArrendamientoFlujoFirmaBaremacion{}).CoincideHuellaHMACSHA256(clave, huella) {
		t.Fatal("una capacidad alterada, ajena o nula supero la autenticacion HMAC")
	}
	if _, err := token.HuellaHMACSHA256([]byte("clave-corta")); !errors.Is(err, ErrArrendamientoFlujoFirmaInvalido) {
		t.Fatalf("clave HMAC corta aceptada: %v", err)
	}
	copia := token
	huellaCopia, err := copia.HuellaHMACSHA256(clave)
	if err != nil || !bytes.Equal(huellaCopia, huella) || !copia.CoincideHuellaHMACSHA256(clave, huella) {
		t.Fatalf("copiar el sobre altero la capacidad: huella=%x error=%v", huellaCopia, err)
	}

	tipo := reflect.TypeOf(token)
	if _, existe := tipo.MethodByName("Revelar"); existe {
		t.Fatal("el token expone un metodo Revelar")
	}
	valor := reflect.ValueOf(token)
	campo := valor.Field(0)
	if tipo.NumField() != 1 || tipo.Field(0).Type.Kind() != reflect.Func ||
		campo.CanInterface() || campo.CanSet() {
		t.Fatalf("representacion reflectible insegura: tipo=%v campos=%d clase=%v accesible=%t mutable=%t",
			tipo, tipo.NumField(), tipo.Field(0).Type.Kind(), campo.CanInterface(), campo.CanSet())
	}
	invocable := true
	func() {
		defer func() {
			if recover() != nil {
				invocable = false
			}
		}()
		campo.Call([]reflect.Value{
			reflect.ValueOf(clave), reflect.Zero(campo.Type().In(1)),
		})
	}()
	if invocable {
		t.Fatal("reflect pudo invocar el cierre privado del token")
	}
	for indice := 0; indice < tipo.NumField(); indice++ {
		if tipo.Field(indice).IsExported() {
			t.Fatalf("el token expone el campo %q", tipo.Field(indice).Name)
		}
	}
}

func TestTokenYArrendamientoFlujoFirmaBloqueanSerializacionYRegistro(t *testing.T) {
	token, err := NuevoTokenArrendamientoFlujoFirmaBaremacion()
	if err != nil {
		t.Fatal(err)
	}
	arrendamiento := ArrendamientoFlujoFirmaBaremacion{
		FlujoRef: "flujo-firma-opaco-001", PropietarioRef: "propietario-opaco-001",
		SecuenciaCercado: 7, ExpiraEn: time.Date(2026, 7, 16, 12, 5, 0, 0, time.UTC), Token: token,
	}
	if err := arrendamiento.Validar(); err != nil {
		t.Fatal(err)
	}
	for indice, valor := range []any{token, arrendamiento, struct {
		XMLName       xml.Name                          `xml:"envoltura"`
		Arrendamiento ArrendamientoFlujoFirmaBaremacion `xml:"arrendamiento"`
	}{Arrendamiento: arrendamiento}} {
		if contenido, err := json.Marshal(valor); contenido != nil ||
			!errors.Is(err, ErrSerializacionArrendamientoProhibida) {
			t.Fatalf("objeto %d admite JSON: contenido=%q error=%v", indice, contenido, err)
		}
	}
	for indice, valor := range []any{token, arrendamiento, struct {
		XMLName       xml.Name                          `xml:"envoltura"`
		Arrendamiento ArrendamientoFlujoFirmaBaremacion `xml:"arrendamiento"`
	}{Arrendamiento: arrendamiento}} {
		var gobCodificado bytes.Buffer
		if err := gob.NewEncoder(&gobCodificado).Encode(valor); err == nil ||
			!strings.Contains(err.Error(), ErrSerializacionArrendamientoProhibida.Error()) {
			t.Fatalf("objeto %d admite gob: contenido=%x error=%v", indice, gobCodificado.Bytes(), err)
		}
		if contenido, err := xml.Marshal(valor); contenido != nil ||
			!errors.Is(err, ErrSerializacionArrendamientoProhibida) {
			t.Fatalf("objeto %d admite XML: contenido=%q error=%v", indice, contenido, err)
		}
	}
	for nombre, prueba := range map[string]func() ([]byte, error){
		"token_texto":           token.MarshalText,
		"token_binario":         token.MarshalBinary,
		"arrendamiento_texto":   arrendamiento.MarshalText,
		"arrendamiento_binario": arrendamiento.MarshalBinary,
	} {
		contenido, err := prueba()
		if contenido != nil || !errors.Is(err, ErrSerializacionArrendamientoProhibida) {
			t.Fatalf("%s admite serializacion: contenido=%q error=%v", nombre, contenido, err)
		}
	}

	var tokenRestaurado TokenArrendamientoFlujoFirmaBaremacion
	var arrendamientoRestaurado ArrendamientoFlujoFirmaBaremacion
	for nombre, prueba := range map[string]func() error{
		"token_json":            func() error { return json.Unmarshal([]byte(`"forjado"`), &tokenRestaurado) },
		"token_texto":           func() error { return tokenRestaurado.UnmarshalText([]byte("forjado")) },
		"token_binario":         func() error { return tokenRestaurado.UnmarshalBinary([]byte("forjado")) },
		"token_gob":             func() error { return tokenRestaurado.GobDecode([]byte("forjado")) },
		"token_xml":             func() error { return xml.Unmarshal([]byte(`<token>forjado</token>`), &tokenRestaurado) },
		"arrendamiento_json":    func() error { return json.Unmarshal([]byte(`{}`), &arrendamientoRestaurado) },
		"arrendamiento_texto":   func() error { return arrendamientoRestaurado.UnmarshalText([]byte("forjado")) },
		"arrendamiento_binario": func() error { return arrendamientoRestaurado.UnmarshalBinary([]byte("forjado")) },
		"arrendamiento_gob":     func() error { return arrendamientoRestaurado.GobDecode([]byte("forjado")) },
		"arrendamiento_xml":     func() error { return xml.Unmarshal([]byte(`<arrendamiento/>`), &arrendamientoRestaurado) },
	} {
		if err := prueba(); !errors.Is(err, ErrSerializacionArrendamientoProhibida) {
			t.Fatalf("%s admite deserializacion: %v", nombre, err)
		}
	}

	for _, caso := range []struct {
		valor    any
		marcador string
	}{
		{valor: token, marcador: "[TOKEN-ARRENDAMIENTO-FLUJO-FIRMA-REDACTADO]"},
		{valor: arrendamiento, marcador: "[ARRENDAMIENTO-FLUJO-FIRMA-REDACTADO]"},
	} {
		formateado := fmt.Sprintf("%v|%#v|%+v|%s|%q", caso.valor, caso.valor, caso.valor, caso.valor, caso.valor)
		if strings.Count(formateado, caso.marcador) != 5 || strings.Contains(formateado, "flujo-firma-opaco-001") {
			t.Fatalf("formateo no redactado: %q", formateado)
		}
	}
	var registro bytes.Buffer
	slog.New(slog.NewJSONHandler(&registro, nil)).Info(
		"prueba", "token", token, "arrendamiento", arrendamiento,
	)
	if !strings.Contains(registro.String(), "TOKEN-ARRENDAMIENTO-FLUJO-FIRMA-REDACTADO") ||
		!strings.Contains(registro.String(), "ARRENDAMIENTO-FLUJO-FIRMA-REDACTADO") ||
		strings.Contains(registro.String(), "flujo-firma-opaco-001") {
		t.Fatalf("slog no redacto capacidades: %s", registro.String())
	}
}

func TestEstadoProtegidoFlujoFirmaBloqueaSerializacionYFormateo(t *testing.T) {
	estado, err := NuevoEstadoProtegidoFlujoFirmaBaremacion(
		AlgoritmoProteccionEstadoAES256GCM,
		"clave-estado-flujo-firma-v1",
		bytes.Repeat([]byte{0x11}, 12),
		bytes.Repeat([]byte{0x22}, 32),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := json.Marshal(estado); !errors.Is(err, ErrSerializacionEstadoFlujoProhibida) {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if _, err := estado.MarshalText(); !errors.Is(err, ErrSerializacionEstadoFlujoProhibida) {
		t.Fatalf("MarshalText() error = %v", err)
	}
	formateado := fmt.Sprintf("%v|%#v|%+v", estado, estado, estado)
	if strings.Contains(formateado, "111111") || strings.Contains(formateado, "222222") ||
		strings.Count(formateado, "[ESTADO-FLUJO-FIRMA-PROTEGIDO]") != 3 {
		t.Fatalf("el formateo filtro el sobre: %q", formateado)
	}

	datos, err := estado.DatosPersistencia()
	if err != nil {
		t.Fatal(err)
	}
	datos.Nonce[0] ^= 0xff
	datos.Cifrado[0] ^= 0xff
	if estado.Validar() != nil {
		t.Fatal("DatosPersistencia devolvio alias mutables sobre el estado")
	}
}

func TestRepresentacionCanonicaFlujoFirmaLigaNonceAEAD(t *testing.T) {
	cifrado := bytes.Repeat([]byte{0x44}, 32)
	estadoA, err := NuevoEstadoProtegidoFlujoFirmaBaremacion(
		AlgoritmoProteccionEstadoAES256GCM,
		"clave-estado-flujo-firma-v1",
		bytes.Repeat([]byte{0x01}, 12),
		cifrado,
	)
	if err != nil {
		t.Fatal(err)
	}
	estadoB, err := NuevoEstadoProtegidoFlujoFirmaBaremacion(
		AlgoritmoProteccionEstadoAES256GCM,
		"clave-estado-flujo-firma-v1",
		bytes.Repeat([]byte{0x02}, 12),
		cifrado,
	)
	if err != nil {
		t.Fatal(err)
	}
	instante := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
	expediente := ExpedienteFlujoFirmaBaremacion{
		FlujoRef: "flujo-firma-001", Version: 1,
		IndiceIdempotenciaHMAC: hmacFlujoFirmaPuertosPrueba("1"),
		HuellaSolicitudHMAC:    hmacFlujoFirmaPuertosPrueba("2"),
		VinculoActorHMAC:       hmacFlujoFirmaPuertosPrueba("3"),
		PerfilActorClave:       "perfil_rrhh",
		ProcesoRef:             "proceso-001",
		SolicitudRef:           "solicitud-001",
		BaremacionMeritoRef:    "baremacion-001",
		DecisionRef:            "decision-001",
		Estado:                 EstadoExpedienteFirmaPreparando,
		EstadoProtegido:        estadoA,
		CreadoEn:               instante,
		ActualizadoEn:          instante,
	}
	canonicaA, err := RepresentacionCanonicaExpedienteFlujoFirmaBaremacion(expediente)
	if err != nil {
		t.Fatal(err)
	}
	expediente.EstadoProtegido = estadoB
	canonicaB, err := RepresentacionCanonicaExpedienteFlujoFirmaBaremacion(expediente)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(canonicaA.Revelar(), canonicaB.Revelar()) {
		t.Fatal("la representacion sellada no ligo el nonce AEAD")
	}
}

func TestReglaComunFlujoFirmaReconciliaYProtegeHistoria(t *testing.T) {
	inicial := expedienteInicialFlujoFirmaPuertosPrueba(t, "flujo-firma-regla-001", 0x31)
	reintento := expedienteInicialFlujoFirmaPuertosPrueba(t, "flujo-firma-regla-002", 0x32)
	reintento.IndiceIdempotenciaHMAC = inicial.IndiceIdempotenciaHMAC
	reintento.HuellaSolicitudHMAC = inicial.HuellaSolicitudHMAC
	reintento.VinculoActorHMAC = inicial.VinculoActorHMAC
	reintento.PerfilActorClave = inicial.PerfilActorClave
	reintento.ProcesoRef = inicial.ProcesoRef
	reintento.SolicitudRef = inicial.SolicitudRef
	reintento.BaremacionMeritoRef = inicial.BaremacionMeritoRef
	reintento.DecisionRef = inicial.DecisionRef
	if !MismaSolicitudInicialFlujoFirmaBaremacion(inicial, reintento) {
		t.Fatal("el reintento semántico dependió de referencia, nonce o instante aleatorios")
	}
	reintento.DecisionRef = "decision-firma-ajena"
	if MismaSolicitudInicialFlujoFirmaBaremacion(inicial, reintento) {
		t.Fatal("la reconciliación aceptó una decisión distinta")
	}

	declarada, err := inicial.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	declarada.Version = 2
	declarada.ActualizadoEn = inicial.ActualizadoEn.Add(time.Second)
	declarada.PuntosControl = append(declarada.PuntosControl, PuntoControlFirmaBaremacion{
		Paso:                  PasoPrepararFirmaBaremacion,
		Estado:                EstadoPuntoControlFirmaDeclarado,
		EfectoRef:             "efecto-preparar-firma-001",
		ClaveIdempotenciaHMAC: hmacFlujoFirmaPuertosPrueba("5"),
		DeclaradoEn:           declarada.ActualizadoEn,
	})
	declarada.SelloEstadoHMAC = hmacFlujoFirmaPuertosPrueba("6")
	if err := declarada.Validar(); err != nil {
		t.Fatalf("fixture de transición inválido: %v", err)
	}
	if err := ValidarTransicionFlujoFirmaBaremacion(inicial, declarada); err != nil {
		t.Fatalf("la regla común rechazó una declaración válida: %v", err)
	}

	for nombre, alterar := range map[string]func(*ExpedienteFlujoFirmaBaremacion){
		"actor": func(e *ExpedienteFlujoFirmaBaremacion) {
			e.VinculoActorHMAC = hmacFlujoFirmaPuertosPrueba("7")
		},
		"creación": func(e *ExpedienteFlujoFirmaBaremacion) {
			e.CreadoEn = e.CreadoEn.Add(time.Nanosecond)
		},
		"salto de versión": func(e *ExpedienteFlujoFirmaBaremacion) {
			e.Version = 3
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			alterada, err := declarada.Clonar()
			if err != nil {
				t.Fatal(err)
			}
			alterar(&alterada)
			alterada.SelloEstadoHMAC = hmacFlujoFirmaPuertosPrueba("8")
			if err := ValidarTransicionFlujoFirmaBaremacion(inicial, alterada); !errors.Is(
				err,
				ErrEstadoFlujoFirmaAlterado,
			) {
				t.Fatalf("mutación histórica aceptada: %v", err)
			}
		})
	}
}

func expedienteInicialFlujoFirmaPuertosPrueba(
	t *testing.T,
	flujoRef string,
	byteCifrado byte,
) ExpedienteFlujoFirmaBaremacion {
	t.Helper()
	estado, err := NuevoEstadoProtegidoFlujoFirmaBaremacion(
		AlgoritmoProteccionEstadoAES256GCM,
		"clave-estado-flujo-firma-regla-v1",
		bytes.Repeat([]byte{byteCifrado}, 12),
		bytes.Repeat([]byte{byteCifrado + 1}, 32),
	)
	if err != nil {
		t.Fatal(err)
	}
	instante := time.Date(2026, time.July, 24, 20, 0, 0, 0, time.UTC)
	expediente := ExpedienteFlujoFirmaBaremacion{
		FlujoRef: flujoRef, Version: 1,
		IndiceIdempotenciaHMAC: hmacFlujoFirmaPuertosPrueba("1"),
		HuellaSolicitudHMAC:    hmacFlujoFirmaPuertosPrueba("2"),
		VinculoActorHMAC:       hmacFlujoFirmaPuertosPrueba("3"),
		PerfilActorClave:       "tecnico_rrhh",
		ProcesoRef:             "proceso-firma-regla-001",
		SolicitudRef:           "solicitud-firma-regla-001",
		BaremacionMeritoRef:    "baremacion-firma-regla-001",
		DecisionRef:            "decision-firma-regla-001",
		Estado:                 EstadoExpedienteFirmaPreparando,
		EstadoProtegido:        estado,
		CreadoEn:               instante,
		ActualizadoEn:          instante,
		SelloEstadoHMAC:        hmacFlujoFirmaPuertosPrueba("4"),
	}
	if err := expediente.Validar(); err != nil {
		t.Fatalf("expediente inicial inválido: %v", err)
	}
	return expediente
}

func hmacFlujoFirmaPuertosPrueba(caracter string) string {
	return "hmac-sha256:flujo_firma_prueba_v1:" + strings.Repeat(caracter, 64)
}
