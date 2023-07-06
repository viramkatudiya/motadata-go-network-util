package networkclient

import (
	"context"
	"github.com/viramkatudiya/motadata-go-sdk/logger"
	. "github.com/viramkatudiya/motadata-go-sdk/motadatatypes"
	"github.com/viramkatudiya/motadata-go-sdk/sdkconstant"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type NetworkClient struct {
	host MotadataStringList

	retryCount MotadataUINT16

	timeout MotadataUINT16

	packetSize MotadataUINT16

	port MotadataUINT16

	logger *logger.Logger
}

var (
	pingMinRTT = "Min RTT (ms)"

	pingRTT = "RTT (ms)"

	pingMaxRTT = "Max RTT (ms)"

	pingSendPackets = "Sent Packets"

	pingReceivedPackets = "Received Packets"

	pingLostPackets = "Lost Packets"

	pingPacketLostPercentage = "Packet Lost (%)"
)

func (networkClient *NetworkClient) GetPingDetails() MotadataMap {

	result := make(MotadataMap)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(networkClient.timeout*4)*time.Second)

	defer cancel()

	var commands = []string{"-c", ToString(networkClient.retryCount), "-b", ToString(networkClient.packetSize), "-t", ToString(networkClient.timeout * 1000), "-q", "-u"}

	commands = append(commands, networkClient.host...)

	cmd := exec.CommandContext(ctx, "fping", commands...)

	stdErr, err := cmd.CombinedOutput()

	if err == nil {

		parsePingResult(networkClient, string(stdErr), result)

	} else {

		if IsNotEmpty(string(stdErr)) {

			parsePingResult(networkClient, string(stdErr), result)

		} else {
			result[sdkconstant.ErrorCode] = "Timed out"

			result[sdkconstant.StatusCode] = sdkconstant.StatusCodeUnreachable
		}

	}

	return result

}

func (networkClient *NetworkClient) Init(host MotadataStringList, context MotadataMap) {

	networkClient.host = host

	networkClient.retryCount = context.GetUINT16Value(sdkconstant.ParamPingRetryCount)

	networkClient.timeout = context.GetUINT16Value(sdkconstant.ParamPingTimeout)

	networkClient.packetSize = context.GetUINT16Value(sdkconstant.ParamPingPacketSize)

}

func (networkClient *NetworkClient) SetPort(context MotadataMap) *NetworkClient {

	if context.Contains(sdkconstant.Port) {

		networkClient.port = context.GetUINT16Value(sdkconstant.Port)
	}

	return networkClient
}

func (networkClient *NetworkClient) SetLogger(loggerObj *logger.Logger) *NetworkClient {

	networkClient.logger = loggerObj

	return networkClient
}

func parsePingResult(newWorkClient *NetworkClient, pingResult string, result MotadataMap) {

	if IsNotEmpty(pingResult) {

		for _, output := range strings.Split(pingResult, "\n") {

			if IsNotEmpty(output) && !strings.Contains(output, "duplicate") && !strings.Contains(output, "ICMP Time Exceeded") {

				metrics := make(MotadataMap)

				var host string

				if IsNotEmpty(output) {

					metrics[sdkconstant.StatusCode] = sdkconstant.StatusCodeUnreachable

					if IndexCheck(output, "min/avg/max") {

						metrics[sdkconstant.StatusCode] = sdkconstant.StatusCodeReachable

					}

					tokens := strings.Split(output, ",")

					if len(tokens) > 0 {

						var rttDetails string

						packetPacketDetails := strings.TrimSpace(tokens[0])

						if len(tokens) > 1 {

							rttDetails = strings.TrimSpace(tokens[1])
						}

						host = strings.TrimSpace(strings.TrimSpace(packetPacketDetails)[:strings.LastIndex(packetPacketDetails, ":")])

						packetPacketDetails = strings.TrimSpace(strings.TrimSpace(packetPacketDetails)[strings.LastIndex(packetPacketDetails, ":"):])

						tokens := strings.Split(packetPacketDetails, "=")

						if len(tokens) > 1 {

							tokens := strings.Split(tokens[1], "/")

							if len(tokens) > 1 {

								metrics[pingSendPackets] = ToInt(strings.TrimSpace(tokens[0]))

								metrics[pingReceivedPackets] = ToInt(strings.TrimSpace(tokens[1]))

								metrics[pingPacketLostPercentage] = ToInt(ToFloat64(strings.TrimSpace(strings.Split(strings.TrimSpace(tokens[2]), "%")[0])))

								metrics[pingLostPackets] = ToInt(metrics.GetINTValue(pingSendPackets) - metrics.GetINTValue(pingReceivedPackets))
							}
						}
						if IsNotEmpty(rttDetails) {

							tokens := strings.Split(rttDetails, "=")

							if len(tokens) > 1 {

								tokens := strings.Split(tokens[1], "/")

								if len(tokens) > 0 {

									metrics[pingMinRTT] = ToInt(ToFloat64(strings.TrimSpace(tokens[0])))

									metrics[pingRTT] = ToInt(ToFloat64(strings.TrimSpace(tokens[1])))

									metrics[pingMaxRTT] = ToInt(ToFloat64(strings.TrimSpace(tokens[2])))
								}
							}
						}
					}

					if IsNotEmpty(host) && metrics.IsNotEmpty() {

						result[host] = metrics
					}
				}
			}
		}
	} else {

		result[sdkconstant.ErrorCode] = "Failed To Get Ping Result"

		newWorkClient.logger.Warning(MotadataString(newWorkClient.host[0]), "Failed to get ping metrics", sdkconstant.ModuleNCM)
	}
}

func (networkClient *NetworkClient) IsHostReachable() MotadataMap {

	result := make(MotadataMap)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(networkClient.timeout*4)*time.Second)

	defer cancel()

	var commands = []string{"-c", ToString(networkClient.retryCount), "-b", ToString(networkClient.packetSize), "-t", ToString(networkClient.timeout * 1000), "-q", "-u"}

	commands = append(commands, networkClient.host...)

	cmd := exec.CommandContext(ctx, "fping", commands...)

	stdErr, err := cmd.CombinedOutput()

	if err != nil {

		result[sdkconstant.ErrorCode] = "Failed To Get Ping Result"

		networkClient.logger.Warning(MotadataString(networkClient.host[0]), "Failed to get ping metrics", sdkconstant.ModuleNCM)

	} else {

		if IsNotEmpty(string(stdErr)) {

			if strings.Contains(string(stdErr), "min/avg/max") {

				result[sdkconstant.StatusCode] = sdkconstant.StatusCodeReachable

			} else {

				result[sdkconstant.StatusCode] = sdkconstant.StatusCodeUnreachable
			}

		} else {
			result[sdkconstant.ErrorCode] = "Timed out"

			result[sdkconstant.StatusCode] = sdkconstant.StatusCodeUnreachable
		}
	}

	return result
}

func (networkClient *NetworkClient) IsPortReachable(portType string) MotadataMap {

	result := make(MotadataMap)

	conn, err := net.DialTimeout(portType, net.JoinHostPort(string(networkClient.host[0]), strconv.Itoa(int(networkClient.port))), 10*time.Second)

	if err != nil {

		result[sdkconstant.StatusCode] = sdkconstant.StatusCodeUnreachable
	} else {

		result[sdkconstant.StatusCode] = sdkconstant.StatusCodeReachable
	}

	if conn != nil {
		defer func(conn net.Conn) {
			err := conn.Close()
			if err != nil {

			}
		}(conn)
	}

	return result
}
