package tools

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"time"
)

var Logger *zap.SugaredLogger

func InitLogger() {
	writeSyncer := getLogWriter()
	encoder := getEncoder()
	core := zapcore.NewCore(encoder, writeSyncer, zapcore.DebugLevel)

	logger := zap.New(core, zap.AddCaller())
	Logger = logger.Sugar()

	Logger.Info()
	WorkDir, _ := os.Getwd() // 项目工作目录
	os.Chmod(WorkDir+"/log/out.log", 0777)
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	//encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05"))
	}
	return zapcore.NewConsoleEncoder(encoderConfig)
}

func getLogWriter() zapcore.WriteSyncer {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   "./log/out.log",
		MaxSize:    100,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   false,
	}
	ws := io.MultiWriter(lumberJackLogger, os.Stdout)
	return zapcore.AddSync(ws)
}
