package domain

import (
	"errors"
	"testing"
)

func TestClasificarResultadoTecnicoSeparaResultadoYNivel(t *testing.T) {
	for _, caso := range []struct {
		resultado CodigoResultadoTecnico
		nivel     NivelResultadoTecnico
	}{
		{ResultadoTecnicoCorrecto, NivelResultadoTecnicoInfo},
		{ResultadoTecnicoCancelado, NivelResultadoTecnicoInfo},
		{ResultadoTecnicoDenegado, NivelResultadoTecnicoWarn},
		{ResultadoTecnicoEntradaInvalida, NivelResultadoTecnicoWarn},
		{ResultadoTecnicoNoDisponible, NivelResultadoTecnicoError},
	} {
		clasificada, err := ClasificarResultadoTecnico(SolicitudResultadoTecnico{
			Resultado:  caso.resultado,
			Componente: ComponenteIncidenciaPostgreSQL,
			Etapa:      EtapaIncidenciaConsulta,
		})
		if err != nil || clasificada.Resultado != caso.resultado || clasificada.Nivel != caso.nivel {
			t.Fatalf("resultado %s: clasificación=%+v error=%v", caso.resultado, clasificada, err)
		}
	}
	for _, caso := range []SolicitudResultadoTecnico{
		{Resultado: "otro", Componente: ComponenteIncidenciaPostgreSQL, Etapa: EtapaIncidenciaConsulta},
		{Resultado: ResultadoTecnicoCorrecto, Componente: "persona:privada", Etapa: EtapaIncidenciaConsulta},
		{Resultado: ResultadoTecnicoCorrecto, Componente: ComponenteIncidenciaPostgreSQL, Etapa: "ruta/privada"},
	} {
		if _, err := ClasificarResultadoTecnico(caso); !errors.Is(err, ErrResultadoTecnicoInvalido) {
			t.Fatalf("resultado ajeno admitido: %v", err)
		}
	}
}
