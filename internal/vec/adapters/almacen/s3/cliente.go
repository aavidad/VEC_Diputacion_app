package s3

import (
	"context"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

// clienteSDK es deliberadamente menor que *s3.Client. Permite probar todos
// los limites del adaptador sin servidor ni credenciales reales.
type clienteSDK interface {
	HeadBucket(context.Context, *awss3.HeadBucketInput, ...func(*awss3.Options)) (*awss3.HeadBucketOutput, error)
	GetBucketVersioning(context.Context, *awss3.GetBucketVersioningInput, ...func(*awss3.Options)) (*awss3.GetBucketVersioningOutput, error)
	GetObjectLockConfiguration(context.Context, *awss3.GetObjectLockConfigurationInput, ...func(*awss3.Options)) (*awss3.GetObjectLockConfigurationOutput, error)
	PutObject(context.Context, *awss3.PutObjectInput, ...func(*awss3.Options)) (*awss3.PutObjectOutput, error)
	HeadObject(context.Context, *awss3.HeadObjectInput, ...func(*awss3.Options)) (*awss3.HeadObjectOutput, error)
	GetObject(context.Context, *awss3.GetObjectInput, ...func(*awss3.Options)) (*awss3.GetObjectOutput, error)
	PutObjectRetention(context.Context, *awss3.PutObjectRetentionInput, ...func(*awss3.Options)) (*awss3.PutObjectRetentionOutput, error)
	GetObjectRetention(context.Context, *awss3.GetObjectRetentionInput, ...func(*awss3.Options)) (*awss3.GetObjectRetentionOutput, error)
	PutObjectLegalHold(context.Context, *awss3.PutObjectLegalHoldInput, ...func(*awss3.Options)) (*awss3.PutObjectLegalHoldOutput, error)
	GetObjectLegalHold(context.Context, *awss3.GetObjectLegalHoldInput, ...func(*awss3.Options)) (*awss3.GetObjectLegalHoldOutput, error)
	DeleteObject(context.Context, *awss3.DeleteObjectInput, ...func(*awss3.Options)) (*awss3.DeleteObjectOutput, error)
}

type presignadorSDK interface {
	PresignPutObject(context.Context, *awss3.PutObjectInput, ...func(*awss3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
}

func nuevoClienteReal(ctx context.Context, configuracionS3 Configuracion) (clienteSDK, presignadorSDK, error) {
	if err := configuracionS3.Validar(); err != nil {
		return nil, nil, err
	}
	clienteHTTP, err := clienteHTTPSeguro(configuracionS3.RutaCA, configuracionS3.RedesPermitidas)
	if err != nil {
		return nil, nil, err
	}
	opciones := []func(*config.LoadOptions) error{
		config.WithRegion(configuracionS3.Region),
		config.WithHTTPClient(clienteHTTP),
	}
	if configuracionS3.AccessKeyID != "" {
		opciones = append(opciones, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			configuracionS3.AccessKeyID, configuracionS3.SecretAccessKey, configuracionS3.SessionToken,
		)))
	}
	configuracionAWS, err := config.LoadDefaultConfig(ctx, opciones...)
	if err != nil {
		return nil, nil, ErrOperacionS3
	}
	cliente := awss3.NewFromConfig(configuracionAWS, func(opciones *awss3.Options) {
		opciones.BaseEndpoint = awsv2.String(configuracionS3.Endpoint)
		opciones.UsePathStyle = configuracionS3.PathStyle
	})
	return cliente, awss3.NewPresignClient(cliente), nil
}

var _ clienteSDK = (*awss3.Client)(nil)
var _ presignadorSDK = (*awss3.PresignClient)(nil)
