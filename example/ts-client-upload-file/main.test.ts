import assert from 'node:assert/strict';
import { once } from 'node:events';
import { createServer } from 'node:http';
import { describe, it } from 'node:test';

import { createFileService } from './main.ts';

describe('createFileService', () => {
  it('uploads_files_with_fetch', async t => {
    let requestURL: string | undefined;
    let received: Promise<FormData> | undefined;
    const result = {
      name: 'example',
      file: { filename: 'test.bin', size: 4 },
      files: [
        { filename: 'first.bin', size: 3 },
        { filename: 'second.bin', size: 3 },
      ],
    };
    const server = createServer((request, response) => {
      requestURL = request.url;
      received = (async () => {
        const chunks: ArrayBuffer[] = [];
        for await (const chunk of request) {
          chunks.push(new Uint8Array(chunk as Uint8Array).buffer);
        }
        const form = await new Response(new Blob(chunks), {
          headers: { 'Content-Type': request.headers['content-type'] ?? '' },
        }).formData();
        response.setHeader('Content-Type', 'application/json');
        response.end(JSON.stringify(result));
        return form;
      })();
    });
    t.after(() => server.close());
    server.listen(0, '127.0.0.1');
    await once(server, 'listening');
    const address = server.address();
    assert.ok(address && typeof address !== 'string');

    const service = createFileService(`http://127.0.0.1:${address.port}`);
    assert.deepEqual(
      await service.uploadFile({
        name: 'example',
        file: new File([new Uint8Array([0, 255, 128, 1])], 'test.bin'),
        files: [
          new File([new Uint8Array([2, 0, 254])], 'first.bin'),
          new File([new Uint8Array([2, 0, 254])], 'second.bin'),
        ],
      }),
      result,
    );
    assert.equal(requestURL, '/files/upload');
    const form = await received;
    assert.ok(form);
    assert.equal(form.get('name'), 'example');
    const file = form.get('file');
    assert.ok(file instanceof File);
    assert.equal(file.name, 'test.bin');
    assert.deepEqual(new Uint8Array(await file.arrayBuffer()), new Uint8Array([0, 255, 128, 1]));
    assert.deepEqual(
      form.getAll('files').map(file => {
        assert.ok(file instanceof File);
        return file.name;
      }),
      ['first.bin', 'second.bin'],
    );
  });
});
