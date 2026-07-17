package domain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"testing"
	"time"
)

const (
	claveMotivoAutorizacionV2Prueba           = "motivo_0123456789abcdef0123456789abcdef"
	claveMotivoAutorizacionV2Alternativa      = "motivo_fedcba9876543210fedcba9876543210"
	referenciaCorrelacionAutorizacionV2Prueba = "correlacion_0123456789abcdef0123456789abcdef"
)

func TestHuellaSolicitudAutorizacionV2LigaMotivoCatalogadoYCapacidadesExactas(t *testing.T) {
	solicitud := solicitudHuellaAutorizacionV2Prueba(t)
	huella, err := HuellaSHA256SolicitudAutorizacionV2(solicitud)
	if err != nil {
		t.Fatalf("huella base: %v", err)
	}

	alterada := solicitud
	datosAlterados, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datosAlterados.ReferenciaMotivo.EntradaClave = claveMotivoAutorizacionV2Alternativa
	alterada = nuevaSolicitudHuellaAutorizacionV2Prueba(t, datosAlterados)
	huellaAlterada, err := HuellaSHA256SolicitudAutorizacionV2(alterada)
	if err != nil || huellaAlterada == huella {
		t.Fatalf("Motivo no quedo ligado: huella=%q err=%v", huellaAlterada, err)
	}

	datosSinActor, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datosSinActor.ContextoActor = ContextoActor{}
	sinActor := nuevaSolicitudHuellaAutorizacionV2Prueba(t, datosSinActor)
	huellaSinActor, err := HuellaSHA256SolicitudAutorizacionV2(sinActor)
	if err != nil || huellaSinActor != huella {
		t.Fatalf("el PEP no pudo reconstruir el compromiso desde el vinculo: huella=%q err=%v", huellaSinActor, err)
	}
}

func TestHuellaSolicitudAutorizacionV2RechazaContextoNoCeroInvalido(t *testing.T) {
	solicitud := solicitudHuellaAutorizacionV2Prueba(t)
	datos, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datos.ContextoActor.ResueltoEn = datos.ContextoActor.ResueltoEn.Add(time.Microsecond)
	if _, err := NuevaSolicitudAutorizacionLigadaV2(datos); !errors.Is(err, ErrSolicitudAutorizacionInvalida) {
		t.Fatalf("contexto no cero invalido aceptado: %v", err)
	}
}

func TestHuellaSolicitudAutorizacionV2NoAdmitePrincipalDeclaradoYComprometeRecurso(t *testing.T) {
	solicitud := solicitudHuellaAutorizacionV2Prueba(t)
	huella, err := HuellaSHA256SolicitudAutorizacionV2(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	tipoDatos := reflect.TypeOf(DatosSolicitudAutorizacionLigadaV2{})
	if _, existe := tipoDatos.FieldByName("Principal"); existe {
		t.Fatal("el contrato nominal admite un Principal declarado")
	}
	if _, existe := tipoDatos.FieldByName("CorrelacionRef"); existe {
		t.Fatal("el contrato nominal admite correlacion V2 como texto")
	}
	campoCorrelacion, existe := tipoDatos.FieldByName("Correlacion")
	if !existe || campoCorrelacion.Type != reflect.TypeOf(ReferenciaCorrelacionAutorizacionV2{}) {
		t.Fatalf("el contrato no exige la capacidad nominal de correlacion: %+v", campoCorrelacion)
	}

	datosOtroPrincipal, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datosOtroPrincipal.ContextoActor.Principal.ID = "per_fedcba9876543210fedcba98"
	if _, err := NuevaSolicitudAutorizacionLigadaV2(datosOtroPrincipal); !errors.Is(err, ErrSolicitudAutorizacionInvalida) {
		t.Fatalf("principal distinto del vinculo aceptado: %v", err)
	}
	datosOtroRecurso, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datosOtroRecurso.Recurso.Referencia = "merito:457"
	otroRecurso := nuevaSolicitudHuellaAutorizacionV2Prueba(t, datosOtroRecurso)
	huellaOtroRecurso, err := HuellaSHA256SolicitudAutorizacionV2(otroRecurso)
	if err != nil || huellaOtroRecurso == huella {
		t.Fatalf("recurso efectivo no altero la huella: huella=%q err=%v", huellaOtroRecurso, err)
	}
}

func TestHuellaMotivoAutorizacionV2ComprometeReferenciaCompleta(t *testing.T) {
	primeraReferencia := referenciaMotivoAutorizacionV2Prueba(claveMotivoAutorizacionV2Prueba)
	primera, err := HuellaSHA256MotivoAutorizacionV2(primeraReferencia)
	if err != nil {
		t.Fatalf("huella motivo: %v", err)
	}
	representacion, err := RepresentacionCanonicaMotivoAutorizacionV2(primeraReferencia)
	if err != nil {
		t.Fatalf("representacion motivo: %v", err)
	}
	esperada := `{"esquema":"vec.autorizacion.motivo.v2.referencia-opaca-catalogada","referencia":{"catalogo_id":"motivos_autorizacion","catalogo_version":3,"catalogo_huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","entrada_clave":"motivo_0123456789abcdef0123456789abcdef"}}`
	if string(representacion) != esperada {
		t.Fatalf("representacion canonica inesperada: %s", representacion)
	}
	suma := sha256.Sum256(representacion)
	if hex.EncodeToString(suma[:]) != primera {
		t.Fatal("la huella publica no procede de la representacion durable")
	}
	representacion[0] ^= 0xff
	repetida, err := RepresentacionCanonicaMotivoAutorizacionV2(primeraReferencia)
	if err != nil || string(repetida) != esperada {
		t.Fatalf("la salida mutable altero una llamada posterior: %v", err)
	}
	segundaReferencia := primeraReferencia
	segundaReferencia.CatalogoVersion++
	segunda, err := HuellaSHA256MotivoAutorizacionV2(segundaReferencia)
	if err != nil || primera == segunda {
		t.Fatalf("motivos distintos sin separacion: primera=%q segunda=%q err=%v", primera, segunda, err)
	}
	if _, err := HuellaSHA256MotivoAutorizacionV2(ReferenciaEntradaCatalogo{}); !errors.Is(err, ErrSolicitudAutorizacionInvalida) {
		t.Fatalf("referencia cero aceptada: %v", err)
	}
	nula := primeraReferencia
	nula.CatalogoHuellaSHA256 = "0000000000000000000000000000000000000000000000000000000000000000"
	if _, err := HuellaSHA256MotivoAutorizacionV2(nula); !errors.Is(err, ErrSolicitudAutorizacionInvalida) {
		t.Fatalf("huella centinela cero aceptada: %v", err)
	}
	if ^uint(0)>>63 == 1 {
		desbordada := primeraReferencia
		versionDesbordada := int64(1) << 31
		desbordada.CatalogoVersion = int(versionDesbordada)
		if _, err := HuellaSHA256MotivoAutorizacionV2(desbordada); !errors.Is(err, ErrSolicitudAutorizacionInvalida) {
			t.Fatalf("version no representable en 386 aceptada: %v", err)
		}
	}
}

func TestReferenciaMotivoAutorizacionV2ValidaPerfilCompleto(t *testing.T) {
	referencia := referenciaMotivoAutorizacionV2Prueba(claveMotivoAutorizacionV2Prueba)
	if !ReferenciaMotivoAutorizacionV2Valida(referencia) {
		t.Fatal("referencia opaca valida rechazada")
	}
	for nombre, mutar := range map[string]func(*ReferenciaEntradaCatalogo){
		"catalogo invalido": func(r *ReferenciaEntradaCatalogo) { r.CatalogoID = "" },
		"huella cero": func(r *ReferenciaEntradaCatalogo) {
			r.CatalogoHuellaSHA256 = "0000000000000000000000000000000000000000000000000000000000000000"
		},
		"clave legible": func(r *ReferenciaEntradaCatalogo) { r.EntradaClave = "consulta_rrhh" },
	} {
		t.Run(nombre, func(t *testing.T) {
			alterada := referencia
			mutar(&alterada)
			if ReferenciaMotivoAutorizacionV2Valida(alterada) {
				t.Fatalf("referencia fuera del perfil V2 aceptada: %+v", alterada)
			}
		})
	}
}

func TestSolicitudAutorizacionLigadaV2NoExponeMotivoYRechazaReferenciaFalsa(t *testing.T) {
	solicitud := solicitudHuellaAutorizacionV2Prueba(t)
	if _, existe := reflect.TypeOf(DatosSolicitudAutorizacionLigadaV2{}).FieldByName("Motivo"); existe {
		t.Fatal("el contrato nominal V2 expone Motivo string")
	}
	datosBase, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	for nombre, mutar := range map[string]func(*DatosSolicitudAutorizacionLigadaV2){
		"posible dato personal": func(s *DatosSolicitudAutorizacionLigadaV2) { s.ReferenciaMotivo.EntradaClave = "dni_12345678z" },
		"referencia cero":       func(s *DatosSolicitudAutorizacionLigadaV2) { s.ReferenciaMotivo = ReferenciaEntradaCatalogo{} },
		"clave legible":         func(s *DatosSolicitudAutorizacionLigadaV2) { s.ReferenciaMotivo.EntradaClave = "consulta_rrhh" },
	} {
		t.Run(nombre, func(t *testing.T) {
			alterada := datosBase
			mutar(&alterada)
			if _, err := NuevaSolicitudAutorizacionLigadaV2(alterada); !errors.Is(err, ErrSolicitudAutorizacionInvalida) {
				t.Fatalf("solicitud V2 no catalogada aceptada: %v", err)
			}
		})
	}
}

func TestHuellaMotivoAutorizacionV2SoloAdmiteClaveOpacaDe128Bits(t *testing.T) {
	for nombre, clave := range map[string]string{
		"codigo humano":           "consulta_gobierno_rrhh",
		"posible DNI":             "dni_12345678z",
		"hexadecimal corto":       "motivo_0123456789abcdef",
		"hexadecimal largo":       "motivo_0123456789abcdef0123456789abcdef00",
		"hexadecimal mayuscula":   "motivo_0123456789ABCDEF0123456789abcdef",
		"caracter no hexadecimal": "motivo_0123456789abcdef0123456789abcdeg",
		"prefijo distinto":        "razon__0123456789abcdef0123456789abcdef",
	} {
		t.Run(nombre, func(t *testing.T) {
			referencia := referenciaMotivoAutorizacionV2Prueba(clave)
			if _, err := HuellaSHA256MotivoAutorizacionV2(referencia); !errors.Is(err, ErrSolicitudAutorizacionInvalida) {
				t.Fatalf("clave no opaca aceptada: clave=%q err=%v", clave, err)
			}
		})
	}
}

func TestSolicitudAutorizacionV2SoloAdmiteCorrelacionOpacaDe128Bits(t *testing.T) {
	if !ReferenciaCorrelacionAutorizacionV2Valida(referenciaCorrelacionAutorizacionV2Prueba) {
		t.Fatal("referencia de correlacion opaca valida rechazada")
	}
	for nombre, referencia := range map[string]string{
		"dato de expediente":      "correlacion_expediente_1234",
		"hexadecimal corto":       "correlacion_0123456789abcdef",
		"hexadecimal largo":       "correlacion_0123456789abcdef0123456789abcdef00",
		"hexadecimal mayuscula":   "correlacion_0123456789ABCDEF0123456789abcdef",
		"caracter no hexadecimal": "correlacion_0123456789abcdef0123456789abcdeg",
		"prefijo distinto":        "corr_0123456789abcdef0123456789abcdef",
	} {
		t.Run(nombre, func(t *testing.T) {
			if ReferenciaCorrelacionAutorizacionV2Valida(referencia) {
				t.Fatalf("correlacion no opaca aceptada: %q", referencia)
			}
			generador := &generadorCorrelacionAutorizacionV2Prueba{valor: referencia}
			if _, err := GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), generador); !errors.Is(err, ErrReferenciaCorrelacionAutorizacionV2Invalida) {
				t.Fatalf("fabrica acepto correlacion invalida: %v", err)
			}
		})
	}
	datos, err := solicitudHuellaAutorizacionV2Prueba(t).Datos()
	if err != nil {
		t.Fatal(err)
	}
	datos.Correlacion = ReferenciaCorrelacionAutorizacionV2{}
	if _, err := NuevaSolicitudAutorizacionLigadaV2(datos); !errors.Is(err, ErrSolicitudAutorizacionInvalida) {
		t.Fatalf("solicitud V2 con capacidad cero aceptada: %v", err)
	}
}

func solicitudHuellaAutorizacionV2Prueba(t *testing.T) SolicitudAutorizacionLigadaV2 {
	t.Helper()
	instante := time.Date(2026, 7, 15, 10, 11, 12, 123_456_000, time.UTC)
	actor := contextoActorVinculoPrueba(t, instante)
	vinculo := vinculoAutenticacionActorV1Prueba(t, instante)
	return nuevaSolicitudHuellaAutorizacionV2Prueba(t, DatosSolicitudAutorizacionLigadaV2{
		ContextoActor: actor, VinculoAutenticacionActor: vinculo,
		ReferenciaMotivo: referenciaMotivoAutorizacionV2Prueba(claveMotivoAutorizacionV2Prueba),
		Accion:           "bolsa.merito.revisar",
		Recurso: RecursoAutorizable{
			Referencia: "merito:456", ModuloID: "bolsa", Tipo: "merito",
			Ambitos:   map[string]string{"provincia": "granada", "unidad": "seleccion"},
			Atributos: map[string]string{"estado": "presentado"},
		},
		Finalidad: "gestion_bolsa",
		Correlacion: referenciaCorrelacionAutorizacionV2ParaPrueba(
			t,
			referenciaCorrelacionAutorizacionV2Prueba,
		),
	})
}

func nuevaSolicitudHuellaAutorizacionV2Prueba(
	t *testing.T,
	datos DatosSolicitudAutorizacionLigadaV2,
) SolicitudAutorizacionLigadaV2 {
	t.Helper()
	solicitud, err := NuevaSolicitudAutorizacionLigadaV2(datos)
	if err != nil {
		t.Fatalf("crear solicitud nominal V2: %v", err)
	}
	return solicitud
}

func referenciaMotivoAutorizacionV2Prueba(clave string) ReferenciaEntradaCatalogo {
	return ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_autorizacion", CatalogoVersion: 3,
		CatalogoHuellaSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		EntradaClave:         clave,
	}
}
