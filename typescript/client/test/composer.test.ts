import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { AdaptComposer, FormComposer, JsonComposer, type Request } from '../src/index.ts';

void describe('AdaptComposer.compose', () => {
  void it('uses form for GET regardless of content type', () => {
    const request: Request = { method: 'GET', path: '/', headers: { 'Content-Type': 'application/json' } };

    new AdaptComposer({ form: new FormComposer(), json: new JsonComposer() }).compose(request, { name: 'Alice' });

    assert.deepEqual(request.query, { name: 'Alice' });
    assert.equal(request.body, undefined);
  });

  void it('uses the composer selected by content type', () => {
    const composer = new AdaptComposer({ form: new FormComposer(), json: new JsonComposer() });

    const formRequest: Request = {
      method: 'POST',
      path: '/',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded; charset=utf-8' },
    };
    composer.compose(formRequest, { name: 'Alice' });
    assert.equal(new TextDecoder().decode(formRequest.body), 'name=Alice');

    const jsonRequest: Request = {
      method: 'POST',
      path: '/',
      headers: { 'content-type': 'APPLICATION/JSON; charset=utf-8' },
    };
    composer.compose(jsonRequest, { name: 'Alice' });
    assert.equal(new TextDecoder().decode(jsonRequest.body), '{"name":"Alice"}');
  });

  void it('rejects unsupported media types', () => {
    const composer = new AdaptComposer({ form: new FormComposer(), json: new JsonComposer() });
    const request: Request = { method: 'POST', path: '/', headers: { 'Content-Type': 'text/plain' } };

    assert.throws(() => composer.compose(request, {}), TypeError);
    assert.equal(request.body, undefined);
  });
});

void describe('FormComposer.compose', () => {
  void it('writes GET values to the query', () => {
    const request: Request = { method: 'GET', path: '/', query: { preserved: 'value' } };

    new FormComposer().compose(request, { name: 'Alice', tags: ['a', 'b'] });

    assert.deepEqual(request.query, { preserved: 'value', name: 'Alice', tags: ['a', 'b'] });
  });
});

void describe('JsonComposer.compose', () => {
  void it('writes JSON bytes to the body', () => {
    const request: Request = { method: 'POST', path: '/' };

    new JsonComposer().compose(request, { name: 'Alice' });

    assert.equal(new TextDecoder().decode(request.body), '{"name":"Alice"}');
  });
});
