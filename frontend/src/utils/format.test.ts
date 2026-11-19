import { describe, expect, it } from 'vitest';
import { fileExtension, formatBytes, formatDuration, formatSpeed, shortHash } from './format';

// These strings are what the user reads about their file; the E2E suite asserts on them too,
// so a change here must be deliberate.
describe('formatBytes', () => {
  it.each<[number, string]>([
    [0, '0 B'], [-5, '0 B'], [NaN, '0 B'], [1, '1 B'], [1023, '1023 B'],
    [1024, '1.00 KB'], [1536, '1.50 KB'], [10 * 1024, '10.0 KB'], [100 * 1024, '100 KB'],
    [5 * 1024 * 1024 + 7, '5.00 MB'], [1024 ** 3, '1.00 GB'], [1024 ** 4, '1.00 TB'], [5 * 1024 ** 5, '5120 TB'],
  ])('%d bytes -> %s', (n, want) => expect(formatBytes(n)).toBe(want));
});

describe('formatSpeed', () => {
  it('formats MB/s as bytes per second', () => {
    expect(formatSpeed(12.5)).toBe('12.5 MB/s');
    expect(formatSpeed(0)).toBe('0 B/s');
  });
});

describe('formatDuration', () => {
  it.each<[number, string]>([
    [0, '0s'], [-3, '0s'], [0.4, '0s'], [45, '45s'], [59.6, '1m 00s'], [60, '1m 00s'], [185, '3m 05s'], [3599, '59m 59s'], [3600, '1h 00m'], [3725, '1h 02m'],
  ])('%d s -> %s', (n, want) => expect(formatDuration(n)).toBe(want));
});

describe('shortHash', () => {
  const sha = 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855';
  it('keeps both ends of a long digest', () => expect(shortHash(sha)).toBe('e3b0c4…52b855'));
  it('honours the edge length', () => expect(shortHash(sha, 8)).toBe('e3b0c442…7852b855'));
  it('leaves short strings alone', () => expect(shortHash('abcdef')).toBe('abcdef'));
});

describe('fileExtension', () => {
  it.each<[string, string]>([
    ['photo.png', 'PNG'], ['archive.tar.gz', 'GZ'], ['README', ''], ['.bashrc', ''], ['trailingdot.', ''], ['', ''],
  ])('%j -> %j', (name, want) => expect(fileExtension(name)).toBe(want));
});
