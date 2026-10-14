package ports

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestComandoPeticionCentroV3SeparaEntradaYClonaCampos(t *testing.T) {
	solicitud := solicitudPeticionPuertoPrueba()
	solicitud.MotivoClave, solicitud.JornadaMinutos = "vacante", 2250
	solicitud.NumeroPersonas, solicitud.PuestoSolicitado = 1, "Puesto publicado"
	necesidad := &domain.DatosNecesidadAlta{
		Esquema: "vec.ct.necesidad_alta.v1", CatalogoRef: "catalogo:ct:necesidades",
		CatalogoVersion: 3, CatalogoHuellaSHA256: strings.Repeat("a", 64),
		CausaClave: solicitud.MotivoClave, Periodo: solicitud.Periodo, JornadaMinutos: 2250,
		Campos: map[string]string{"puesto_codigo": "9", "rpt_catalogo_ref": "rpt:publicada"},
	}
	comando := ComandoPeticionCentro{Operacion: OperacionPresentarPeticionCentro,
		ClaveIdempotencia: "12345678-1234-4234-8234-123456789abc",
		EsquemaAlta:       EsquemaAltaNecesidadV1, NecesidadEntrada: necesidad, Solicitud: solicitud}
	clon, err := comando.Clonar()
	if err != nil || clon.NecesidadEntrada == nil || clon.EsquemaAlta != EsquemaAltaNecesidadV1 {
		t.Fatalf("entrada v3 inválida: %v", err)
	}
	comando.NecesidadEntrada.Campos["puesto_codigo"] = "otro"
	if clon.NecesidadEntrada.Campos["puesto_codigo"] != "9" {
		t.Fatal("la entrada comparte campos mutables")
	}
	conInstantanea := clon
	copia := *clon.NecesidadEntrada
	copia.CatalogoInstantanea = []byte("cliente")
	conInstantanea.NecesidadEntrada = &copia
	if conInstantanea.Validar() == nil {
		t.Fatal("el cliente aportó una instantánea de servidor")
	}
	cruzada := clon
	copia = *clon.NecesidadEntrada
	copia.CausaClave = "sustitucion"
	cruzada.NecesidadEntrada = &copia
	if cruzada.Validar() == nil {
		t.Fatal("causa de necesidad distinta de la solicitud")
	}
}

func TestContratoPeticionCentroClonaSolicitudYNoExponeActorEnComando(t *testing.T) {
	comando := ComandoPeticionCentro{Operacion: OperacionPresentarPeticionCentro,
		ClaveIdempotencia: "12345678-1234-4234-8234-123456789abc", Solicitud: solicitudPeticionPuertoPrueba()}
	clon, err := comando.Clonar()
	if err != nil || clon.ClaveIdempotencia != comando.ClaveIdempotencia {
		t.Fatalf("clonar comando y UUID: %#v, %v", clon, err)
	}
	comando.Solicitud.DocumentosAdjuntos[0] = "documento:alterado"
	if clon.Solicitud.DocumentosAdjuntos[0] != "documento:001" {
		t.Fatal("el clon comparte documentos con el comando")
	}
	tipo := reflect.TypeOf(ComandoPeticionCentro{})
	for _, nombre := range []string{"Actor", "ActorRef", "PerfilRef", "PuestoRef"} {
		if _, existe := tipo.FieldByName(nombre); existe {
			t.Fatalf("el actor autenticado aparece en el comando: %s", nombre)
		}
	}
}

func TestMaterialYReciboPeticionCentroQuedanLigados(t *testing.T) {
	ahora := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	solicitud := solicitudPeticionPuertoPrueba()
	config := configuracionPeticionPuertoPrueba()
	peticion, err := domain.NuevaPeticionCentro("peticion:centro:12345678-1234-4234-8234-123456789abc", config, *solicitud, ahora)
	if err != nil {
		t.Fatal(err)
	}
	material := MaterialPeticionCentro{Comando: ComandoPeticionCentro{Operacion: OperacionPresentarPeticionCentro,
		ClaveIdempotencia: "12345678-1234-4234-8234-123456789abc", Solicitud: solicitud},
		Actor: config.Solicitante, Peticion: peticion.Datos()}
	recibo := ReciboPeticionCentro{ReciboRef: "recibo:peticion:001", PeticionRef: material.Peticion.Referencia,
		Version: 1, Estado: "pendiente_ratificacion", ActorRef: config.Solicitante.ActorRef,
		RegistradoEn: ahora.Add(time.Minute), EstadoLocal: "registrado"}
	if err := material.Validar(); err != nil {
		t.Fatalf("material válido: %v", err)
	}
	if err := recibo.ValidarPara(material); err != nil {
		t.Fatalf("recibo válido: %v", err)
	}
	recibo.PeticionRef = "peticion:centro:otra"
	if err := recibo.ValidarPara(material); err != ErrReciboPeticionCentroNoConfiable {
		t.Fatalf("recibo desligado aceptado: %v", err)
	}
}

func solicitudPeticionPuertoPrueba() *domain.SolicitudCentro {
	inicio := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	return &domain.SolicitudCentro{CentroRef: "centro:sintetico:001", ContactoRef: "contacto:sintetico:001",
		CategoriaRef: "categoria:tecnica", GrupoSubgrupo: "A1", MotivoClave: "necesidad.temporal",
		Detalle: "Necesidad sintética", Periodo: domain.PeriodoPrevisto{Inicio: inicio, Fin: inicio.Add(24 * time.Hour)},
		DocumentosAdjuntos: []string{"documento:001"}}
}

func configuracionPeticionPuertoPrueba() domain.ConfiguracionPeticionCentro {
	return domain.ConfiguracionPeticionCentro{Referencia: "config:peticion:001", Version: 1,
		Solicitante: domain.ActorPeticionCentro{ActorRef: "actor:solicitante", PerfilRef: "perfil:responsable", CentroRef: "centro:sintetico:001", PuestoRef: "puesto:jefatura"},
		Ratificador: domain.ActorPeticionCentro{ActorRef: "actor:ratificador", PerfilRef: "perfil:responsable", CentroRef: "centro:sintetico:001", PuestoRef: "puesto:direccion"}}
}
