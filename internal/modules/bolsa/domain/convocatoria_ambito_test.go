package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestAmbitoOrganizativoConvocatoriaExigeReferenciasCanonicasSinDefaults(t *testing.T) {
	valido, err := NuevoAmbitoOrganizativoConvocatoria(
		"org_diputaciongranada", "uni_seleccionexterna",
	)
	if err != nil || valido.OrganizacionRef() != "org_diputaciongranada" ||
		valido.UnidadGestionRef() != "uni_seleccionexterna" {
		t.Fatalf("ambito valido rechazado: %#v / %v", valido, err)
	}
	organizativo, err := NuevoAmbitoOrganizativoConvocatoria("org_diputaciongranada", "")
	if err != nil || organizativo.UnidadGestionRef() != "" {
		t.Fatalf("ambito de organizacion valido rechazado: %#v / %v", organizativo, err)
	}

	casos := []struct {
		nombre       string
		organizacion string
		unidad       string
	}{
		{nombre: "cero"},
		{nombre: "organizacion ausente", unidad: "uni_seleccionexterna"},
		{nombre: "prefijo organizacion", organizacion: "uni_diputaciongranada"},
		{nombre: "carga organizacion corta", organizacion: "org_corta"},
		{nombre: "mayuscula", organizacion: "org_Diputaciongranada"},
		{nombre: "espacio", organizacion: " org_diputaciongranada"},
		{nombre: "comodin", organizacion: "org_diputaciongranad*"},
		{nombre: "prefijo unidad", organizacion: "org_diputaciongranada", unidad: "org_seleccionexterna"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := NuevoAmbitoOrganizativoConvocatoria(
				caso.organizacion, caso.unidad,
			); !errors.Is(err, ErrVersionConvocatoriaGobernadaInvalida) {
				t.Fatalf("ambito no canonico aceptado: %v", err)
			}
		})
	}
}

func TestNuevaVersionNoInterpretaAmbitoCeroComoGlobal(t *testing.T) {
	valida := versionConvocatoriaGobernadaPrueba(t)
	datos := DatosNuevaVersionConvocatoriaGobernada{
		ID: valida.ID, CodigoVersionPublica: valida.CodigoVersionPublica,
		InstanciaFlujoRef: valida.InstanciaFlujoRef,
		Contenido:         valida.Contenido, Configuracion: valida.Configuracion,
		ExpedienteRef: valida.ExpedienteRef, Motivo: valida.MotivoCreacion,
		ActorID: valida.CreadaPor, Instante: valida.CreadaEn,
	}
	if _, err := NuevaVersionConvocatoriaGobernada(datos); !errors.Is(
		err, ErrVersionConvocatoriaGobernadaInvalida,
	) {
		t.Fatalf("ambito cero interpretado como acceso global: %v", err)
	}
}

func TestAmbitoOrganizativoFormaParteDeAmbasHuellasYDeLosBytes(t *testing.T) {
	original := versionConvocatoriaGobernadaPrueba(t)
	otroAmbito, err := NuevoAmbitoOrganizativoConvocatoria(
		"org_diputaciongranada", "uni_gestionrecursoshumanos",
	)
	if err != nil {
		t.Fatal(err)
	}
	modificada := original
	modificada.AmbitoOrganizativo = otroAmbito
	modificada, err = modificada.ClonarCanonico()
	if err != nil {
		t.Fatal(err)
	}

	contenidoOriginal, _ := original.RepresentacionContenidoCanonica()
	contenidoModificado, _ := modificada.RepresentacionContenidoCanonica()
	estadoOriginal, _ := original.RepresentacionCanonica()
	estadoModificado, _ := modificada.RepresentacionCanonica()
	huellaContenidoOriginal, _ := original.HuellaContenidoSHA256()
	huellaContenidoModificada, _ := modificada.HuellaContenidoSHA256()
	huellaEstadoOriginal, _ := original.HuellaSHA256()
	huellaEstadoModificada, _ := modificada.HuellaSHA256()
	if bytes.Equal(contenidoOriginal, contenidoModificado) || bytes.Equal(estadoOriginal, estadoModificado) ||
		huellaContenidoOriginal == huellaContenidoModificada || huellaEstadoOriginal == huellaEstadoModificada {
		t.Fatal("el cambio de ambito no altero todos los compromisos canonicos")
	}
	if !bytes.Contains(contenidoOriginal, []byte(`"ambito_organizativo":{"organizacion_ref":"org_diputaciongranada","unidad_gestion_ref":"uni_seleccionexterna"}`)) {
		t.Fatalf("contenido canonico sin ambito exacto: %s", contenidoOriginal)
	}
	serializado, err := json.Marshal(original.AmbitoOrganizativo)
	if err != nil || string(serializado) != `{"organizacion_ref":"org_diputaciongranada","unidad_gestion_ref":"uni_seleccionexterna"}` {
		t.Fatalf("representacion JSON del valor inmutable: %s / %v", serializado, err)
	}
}

func TestSucesionHeredaAmbitoExactoYRechazaCambioImplicito(t *testing.T) {
	primera := versionConvocatoriaGobernadaPrueba(t)
	primera = publicarVersionConvocatoriaPrueba(t, primera, primera.CreadaEn.Add(time.Hour))
	sucesora, err := primera.NuevaVersion(
		"v2", primera.Contenido, primera.Configuracion, "expediente:seleccion:2026-002",
		"persona:tecnica:003", "Correccion de bases.", primera.PublicadaEn.Add(time.Hour),
	)
	if err != nil || sucesora.AmbitoOrganizativo != primera.AmbitoOrganizativo {
		t.Fatalf("sucesora sin herencia exacta: %#v / %v", sucesora.AmbitoOrganizativo, err)
	}
	otroAmbito, err := NuevoAmbitoOrganizativoConvocatoria(
		"org_diputaciongranada", "uni_gestionrecursoshumanos",
	)
	if err != nil {
		t.Fatal(err)
	}
	sucesora.AmbitoOrganizativo = otroAmbito
	sucesora, err = sucesora.ClonarCanonico()
	if err != nil {
		t.Fatal(err)
	}
	fecha := sucesora.CreadaEn.Add(time.Hour)
	aprobacion, dependencias := evidenciasPublicacionPrueba(t, sucesora, fecha)
	if _, err := sucesora.PublicarSucesora(
		primera, "persona:gestora:002", aprobacion, dependencias, "Cambio implicito.", fecha,
	); !errors.Is(err, ErrTransicionGobiernoConvocatoria) {
		t.Fatalf("la sucesion acepto cambiar el ambito: %v", err)
	}
}
