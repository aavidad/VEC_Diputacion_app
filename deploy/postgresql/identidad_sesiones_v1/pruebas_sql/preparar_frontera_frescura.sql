SELECT cuenta_ref
  FROM vec_identidad_sesiones_v1.provisionar_cuenta_v1(
      'opr_rrrrrrrrrrrrrrrrrrrrrrrr',
      'vec.identidad.hmac-sha256.v1',
      'idh_aaaaaaaaaaaaaaaaaaaaaaaa', 'clave-hsm-prueba', 1,
      decode(repeat('ab', 32), 'hex'),
      decode(repeat('ac', 32), 'hex'), false, NULL
  );
