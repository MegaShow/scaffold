import process from 'node:process';

import { AdaptComposer, FetchClient, JsonResolver } from '@icytown/scaffold-client';

import { FileService } from './gen/file.service.ts';

export function createFileService(baseURL = 'http://localhost:3000'): FileService {
  const client = new FetchClient({ baseURL });
  const composer = new AdaptComposer();
  const resolver = new JsonResolver();
  return new FileService(client, { composer, resolver });
}

async function main() {
  const service = createFileService();
  const result = await service.uploadFile({
    name: 'example',
    file: new File([new Uint8Array([0, 255, 128, 1])], 'test.bin'),
    files: [
      new File([new Uint8Array([2, 0, 254, 1, 2])], 'first.bin'),
      new File([new Uint8Array([2, 0, 254])], 'second.bin'),
    ],
  });
  process.stdout.write(`${JSON.stringify(result)}\n`);
}

if (import.meta.main) {
  await main();
}
