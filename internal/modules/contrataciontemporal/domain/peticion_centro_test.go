package domain

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestPeticionCentroJSONPendienteSinFechaDeRatificacion(t *testing.T) {
	config, solicitud, creada := datosPeticionCentroPrueba()
	solicitud.DocumentosAdjuntos = nil
	p, err := NuevaPeticionCentro("peticion:centro:001", config, solicitud, creada)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(p.Datos())
	if err != nil {
		t.Fatal(err)
	}
	var campos map[string]json.RawMessage
	if json.Unmarshal(b, &campos) != nil || len(campos) != 6 || campos["ratificada_en"] != nil {
		t.Fatalf("contrato pendiente incompatible con PostgreSQL: %s", b)
	}
	var datos DatosPeticionCentro
	if json.Unmarshal(b, &datos) != nil {
		t.Fatal("JSON inválido")
	}
	if _, err := RehidratarPeticionCentro(datos); err != nil {
		t.Fatal(err)
	}
}

func TestPeticionCentroCicloYCopiasDefensivas(t *testing.T) {
	config, solicitud, creada := datosPeticionCentroPrueba()
	peticion, err := NuevaPeticionCentro("peticion:centro:001", config, solicitud, creada)
	if err != nil {
		t.Fatalf("crear petición: %v", err)
	}
	solicitud.DocumentosAdjuntos[0] = "documento:alterado"
	inicial := peticion.Datos()
	if inicial.Version != 1 || inicial.Estado != "pendiente_ratificacion" ||
		inicial.Solicitud.DocumentosAdjuntos[0] != "documento:001" {
		t.Fatalf("estado inicial o copia incorrectos: %#v", inicial)
	}
	inicial.Solicitud.DocumentosAdjuntos[0] = "documento:mutado"
	if peticion.Datos().Solicitud.DocumentosAdjuntos[0] != "documento:001" {
		t.Fatal("Datos comparte la lista mutable de documentos")
	}

	ratificada, err := peticion.Ratificar(config.Ratificador, 1, "Necesidad confirmada", creada.Add(time.Minute))
	if err != nil {
		t.Fatalf("ratificar: %v", err)
	}
	final := ratificada.Datos()
	if final.Version != 2 || final.Estado != "ratificada" ||
		final.MotivoRatificacion != "Necesidad confirmada" ||
		!reflect.DeepEqual(final.Solicitud, peticion.Datos().Solicitud) {
		t.Fatalf("ratificación incorrecta: %#v", final)
	}
	rehidratada, err := RehidratarPeticionCentro(final)
	if err != nil || !reflect.DeepEqual(rehidratada.Datos(), final) {
		t.Fatalf("rehidratar: datos=%#v error=%v", rehidratada.Datos(), err)
	}
	if _, err := ratificada.Ratificar(config.Ratificador, 2, "Otra", creada.Add(2*time.Minute)); !errors.Is(err, ErrRatificacionCentroDenegada) {
		t.Fatalf("segunda ratificación: %v", err)
	}
}

func TestPeticionCentroDeniegaConfiguracionActorVersionYTiempoInvalidos(t *testing.T) {
	config, solicitud, creada := datosPeticionCentroPrueba()
	invalidas := []ConfiguracionPeticionCentro{config, config, config}
	invalidas[0].Ratificador.ActorRef = config.Solicitante.ActorRef
	invalidas[0].Ratificador.PerfilRef = "perfil:distinto"
	invalidas[1].Ratificador.CentroRef = "centro:otro"
	invalidas[2].Version = 0
	for _, invalida := range invalidas {
		if _, err := NuevaPeticionCentro("peticion:centro:001", invalida, solicitud, creada); !errors.Is(err, ErrPeticionCentroInvalida) {
			t.Fatalf("configuración inválida aceptada: %#v, error=%v", invalida, err)
		}
	}
	peticion, _ := NuevaPeticionCentro("peticion:centro:001", config, solicitud, creada)
	if _, err := peticion.Ratificar(config.Ratificador, 2, "Motivo", creada); !errors.Is(err, ErrVersionPeticionCentroEnConflicto) {
		t.Fatalf("versión incorrecta: %v", err)
	}
	for nombre, cambiar := range map[string]func(*ActorPeticionCentro){
		"actor":  func(a *ActorPeticionCentro) { a.ActorRef = "actor:otro" },
		"perfil": func(a *ActorPeticionCentro) { a.PerfilRef = "perfil:otro" },
		"centro": func(a *ActorPeticionCentro) { a.CentroRef = "centro:otro" },
		"puesto": func(a *ActorPeticionCentro) { a.PuestoRef = "puesto:otro" },
	} {
		actor := config.Ratificador
		cambiar(&actor)
		if _, err := peticion.Ratificar(actor, 1, "Motivo", creada); !errors.Is(err, ErrRatificacionCentroDenegada) {
			t.Fatalf("%s distinto: %v", nombre, err)
		}
	}
	for _, caso := range []struct {
		motivo string
		ahora  time.Time
	}{{"", creada}, {strings.Repeat("a", 1001), creada}, {strings.Repeat("á", 501), creada}, {"Motivo\x00", creada}, {"Motivo\tno canónico", creada}, {" Motivo", creada}, {"Motivo", creada.Add(-time.Microsecond)}, {"Motivo", creada.In(time.FixedZone("CET", 3600))}} {
		if _, err := peticion.Ratificar(config.Ratificador, 1, caso.motivo, caso.ahora); !errors.Is(err, ErrRatificacionCentroDenegada) {
			t.Fatalf("entrada de ratificación inválida aceptada: %#v, error=%v", caso, err)
		}
	}
}

func TestPeticionCentroRehidratacionRechazaEstadosAdulterados(t *testing.T) {
	config, solicitud, creada := datosPeticionCentroPrueba()
	base := DatosPeticionCentro{Referencia: "peticion:centro:001", Version: 1, Configuracion: config,
		Solicitud: solicitud, Estado: "pendiente_ratificacion", CreadaEn: creada}
	casos := []DatosPeticionCentro{base, base, base, base}
	casos[0].Version = 2
	casos[1].Estado = "ratificada"
	casos[1].Version = 2
	casos[1].RatificadaEn = creada.Add(-time.Microsecond)
	casos[1].MotivoRatificacion = "Motivo"
	casos[2].Solicitud.CentroRef = "centro:otro"
	casos[3].Estado = "desconocido"
	for _, datos := range casos {
		if _, err := RehidratarPeticionCentro(datos); !errors.Is(err, ErrPeticionCentroInvalida) {
			t.Fatalf("estado adulterado aceptado: %#v, error=%v", datos, err)
		}
	}
}

func datosPeticionCentroPrueba() (ConfiguracionPeticionCentro, SolicitudCentro, time.Time) {
	centro := "centro:sintetico:001"
	config := ConfiguracionPeticionCentro{Referencia: "config:peticion:001", Version: 3,
		Solicitante: ActorPeticionCentro{"actor:solicitante", "perfil:responsable", centro, "puesto:jefatura"},
		Ratificador: ActorPeticionCentro{"actor:ratificador", "perfil:responsable", centro, "puesto:direccion"}}
	creada := time.Date(2026, 9, 6, 10, 0, 0, 123000000, time.UTC)
	solicitud := SolicitudCentro{CentroRef: centro, ContactoRef: "contacto:sintetico:001",
		CategoriaRef: "categoria:tecnica", GrupoSubgrupo: "A1", MotivoClave: "necesidad.temporal",
		Detalle: "Necesidad sintética", Periodo: PeriodoPrevisto{Inicio: creada.Truncate(24 * time.Hour), Fin: creada.Truncate(24 * time.Hour).Add(24 * time.Hour)},
		DocumentosAdjuntos: []string{"documento:001"}}
	return config, solicitud, creada
}
