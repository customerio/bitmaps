# bitmaps

This has two bitmap implementations capable of holding a fixed size bitmap of nbits.

## Fixed

Fixed is a fixed size bitmap stored an array of uint64 capable of
holding nbits of data.

## Boring

Boring is a single roaring bitmap container capable of holding nbits
of data. The data is held either as an array of uint16 (for each
bit set), or as fixed size bitmap of uint64.

The implementation always allocates a fixed sized buffer which is either used to store the array list, or is used to store the bitmap. The buffer is never reallocated.

One the array data grows over half the container the container is switched to a bitmap form.

The implementation is pretty complicated because it must be capable doing all operations with both bitmaps and array lists.

## Marshaled format

Both bitmap implementation support the same marshalled format, which is

```
8 byte header | data
```

The header is:
```
4 byte magic number
1 byte encoding
1 byte padding
2 byte cardinality
```

The data is either an array of uint16, or nbits of encoded bitmap.

The marshalled format is not portable, and is encoded in whatever
the native endian-ness of the host.
