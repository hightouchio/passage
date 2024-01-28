import * as grpc from "@grpc/grpc-js";
import * as protoLoader from "@grpc/proto-loader";
import {ProtoGrpcType} from "../proto/service";
import {GetTunnelResponse__Output} from "../proto/GetTunnelResponse";

const packageDefinition = protoLoader.loadSync('../../proto/service.proto');
const proto = grpc.loadPackageDefinition(
    packageDefinition
) as unknown as ProtoGrpcType;

void (async () => {
    const client = new proto.Passage("localhost:8081", grpc.credentials.createInsecure());

    // const tunnel = await new Promise<Tunnel__Output | undefined>((resolve, reject) => {
    //     client.createReverseTunnel({
    //         createKeyPair: true,
    //     }, (err, value) => {
    //         if (err != null) reject(err);
    //         else resolve(value);
    //     });
    // });
    // console.log("Create tunnel", tunnel);

    const tunnel = await new Promise<GetTunnelResponse__Output | undefined>((resolve, reject) => {
        client.getTunnel({
            id: "daff9cc0-6eb8-4042-9da9-ffc4fb0eabe0",
        }, (err, value) => {
            if (err != null) reject(err);
            else resolve(value);
        });
    });
    console.log("Tunnel", tunnel);
})();