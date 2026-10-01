use std::collections::BTreeMap;

const MAX_DEPTH: usize = 100;
const MAX_VALUE_COUNT: usize = 16_000_000;
mod scg;

pub(crate) use scg::ScgGeometry;

#[derive(Debug, Clone)]
pub(crate) enum KaValue {
    Null,
    Bool(bool),
    Signed(i64),
    Unsigned(u64),
    Float(f64),
    String(String),
    Bytes(Vec<u8>),
    Array(Vec<KaValue>),
    Map(BTreeMap<String, KaValue>),
}

impl KaValue {
    pub(crate) fn get(&self, key: &str) -> Option<&KaValue> {
        match self {
            Self::Map(map) => map.get(key),
            _ => None,
        }
    }

    pub(crate) fn as_str(&self) -> Option<&str> {
        match self {
            Self::String(value) => Some(value),
            _ => None,
        }
    }

    pub(crate) fn as_bytes(&self) -> Option<&[u8]> {
        match self {
            Self::Bytes(value) => Some(value),
            _ => None,
        }
    }

    pub(crate) fn as_f64(&self) -> Option<f64> {
        match self {
            Self::Float(value) => Some(*value),
            Self::Signed(value) => Some(*value as f64),
            Self::Unsigned(value) => Some(*value as f64),
            _ => None,
        }
    }

    pub(crate) fn as_u64(&self) -> Option<u64> {
        match self {
            Self::Unsigned(value) => Some(*value),
            Self::Signed(value) if *value >= 0 => Some(*value as u64),
            _ => None,
        }
    }

    pub(crate) fn as_bool(&self) -> bool {
        match self {
            Self::Bool(value) => *value,
            Self::Signed(value) => *value != 0,
            Self::Unsigned(value) => *value != 0,
            _ => false,
        }
    }

    pub(crate) fn as_array(&self) -> Option<&[KaValue]> {
        match self {
            Self::Array(value) => Some(value),
            _ => None,
        }
    }

    pub(crate) fn as_map(&self) -> Option<&BTreeMap<String, KaValue>> {
        match self {
            Self::Map(value) => Some(value),
            _ => None,
        }
    }
}

pub(crate) fn parse_sc2_scene(data: &[u8]) -> Result<KaValue, String> {
    let mut reader = Reader::new(data);
    if reader.take(4)? != b"SFV2" {
        return Err("SC2 input does not begin with SFV2".to_owned());
    }
    let _version = reader.u32()?;
    let _node_count = reader.u32()?;
    let _version_tags = reader.archive(None, 0)?;
    let descriptor_length = reader.u32()? as usize;
    reader.take(descriptor_length)?;
    let root = reader.archive(None, 0)?;
    if reader.remaining() != 0 {
        return Err(format!("unparsed SC2 bytes remain: {}", reader.remaining()));
    }
    Ok(root)
}

pub(crate) fn parse_scg(data: &[u8]) -> Result<ScgGeometry, String> {
    let mut reader = Reader::new(data);
    if reader.take(4)? != b"SCPG" {
        return Err("SCG input does not begin with SCPG".to_owned());
    }
    let version = reader.u32()?;
    let count = reader.u32()? as usize;
    let count_copy = reader.u32()? as usize;
    if version != 1 || count != count_copy || count > MAX_VALUE_COUNT {
        return Err(format!(
            "unsupported SCG header: version={version}, counts={count}/{count_copy}"
        ));
    }
    let mut groups = Vec::with_capacity(count);
    for _ in 0..count {
        groups.push(reader.archive(None, 0)?);
    }
    if reader.remaining() != 0 {
        return Err(format!("unparsed SCG bytes remain: {}", reader.remaining()));
    }
    ScgGeometry::from_groups(groups)
}

struct Reader<'a> {
    data: &'a [u8],
    position: usize,
}

impl<'a> Reader<'a> {
    fn new(data: &'a [u8]) -> Self {
        Self { data, position: 0 }
    }

    fn remaining(&self) -> usize {
        self.data.len().saturating_sub(self.position)
    }

    fn take(&mut self, length: usize) -> Result<&'a [u8], String> {
        let end = self
            .position
            .checked_add(length)
            .ok_or_else(|| "binary read offset overflow".to_owned())?;
        if end > self.data.len() {
            return Err(format!(
                "read outside file: offset={}, size={length}, length={}",
                self.position,
                self.data.len()
            ));
        }
        let start = self.position;
        self.position = end;
        Ok(&self.data[start..end])
    }

    fn u8(&mut self) -> Result<u8, String> {
        Ok(self.take(1)?[0])
    }

    fn u16(&mut self) -> Result<u16, String> {
        Ok(u16::from_le_bytes(self.take(2)?.try_into().unwrap()))
    }

    fn u32(&mut self) -> Result<u32, String> {
        Ok(u32::from_le_bytes(self.take(4)?.try_into().unwrap()))
    }

    fn i32(&mut self) -> Result<i32, String> {
        Ok(i32::from_le_bytes(self.take(4)?.try_into().unwrap()))
    }

    fn f32(&mut self) -> Result<f32, String> {
        Ok(f32::from_le_bytes(self.take(4)?.try_into().unwrap()))
    }

    fn string(&mut self) -> Result<String, String> {
        let length = self.u32()? as usize;
        if length > self.remaining() {
            return Err(format!(
                "string length {length} exceeds remaining archive bytes"
            ));
        }
        Ok(String::from_utf8_lossy(self.take(length)?).into_owned())
    }

    fn archive(
        &mut self,
        parent_fast_names: Option<&BTreeMap<u32, String>>,
        depth: usize,
    ) -> Result<KaValue, String> {
        if depth > MAX_DEPTH {
            return Err("SC2 keyed archives exceed the nesting limit".to_owned());
        }
        if self.take(2)? != b"KA" {
            return Err(format!(
                "invalid keyed archive signature at byte {}",
                self.position - 2
            ));
        }
        let version = self.u16()?;
        match version {
            1 => {
                let count = self.count()?;
                let mut map = BTreeMap::new();
                for _ in 0..count {
                    let key_type = self.u8()?;
                    let key = self.value(key_type, parent_fast_names, depth + 1, false)?;
                    let value_type = self.u8()?;
                    let value = self.value(value_type, parent_fast_names, depth + 1, false)?;
                    map.insert(value_key(&key), value);
                }
                Ok(KaValue::Map(map))
            }
            2 => {
                let fast_count = self.count()?;
                let mut names_by_id = BTreeMap::new();
                let mut names = Vec::with_capacity(fast_count);
                for _ in 0..fast_count {
                    let length = self.u16()? as usize;
                    names.push(String::from_utf8_lossy(self.take(length)?).into_owned());
                }
                let mut ids = Vec::with_capacity(fast_count);
                for _ in 0..fast_count {
                    ids.push(self.u32()?);
                }
                for (id, name) in ids.into_iter().zip(names) {
                    names_by_id.insert(id, name);
                }
                let count = self.count()?;
                let mut map = BTreeMap::new();
                for _ in 0..count {
                    let key_id = self.u32()?;
                    let key = names_by_id
                        .get(&key_id)
                        .ok_or_else(|| format!("unknown SC2 fast-name key id {key_id}"))?
                        .clone();
                    let value_type = self.u8()?;
                    let value = self.value(value_type, Some(&names_by_id), depth + 1, false)?;
                    map.insert(key, value);
                }
                Ok(KaValue::Map(map))
            }
            0x0102 => {
                let names = parent_fast_names.ok_or_else(|| {
                    "nested SC2 archive is missing its fast-name table".to_owned()
                })?;
                let count = self.count()?;
                let mut map = BTreeMap::new();
                for _ in 0..count {
                    let key_id = self.u32()?;
                    let key = names
                        .get(&key_id)
                        .ok_or_else(|| format!("unknown nested SC2 key id {key_id}"))?
                        .clone();
                    let value_type = self.u8()?;
                    let value = self.value(value_type, Some(names), depth + 1, true)?;
                    map.insert(key, value);
                }
                Ok(KaValue::Map(map))
            }
            0xff02 => Ok(KaValue::Map(BTreeMap::new())),
            other => Err(format!("unsupported keyed archive version 0x{other:04x}")),
        }
    }

    fn count(&mut self) -> Result<usize, String> {
        let count = self.u32()? as usize;
        if count > MAX_VALUE_COUNT {
            return Err(format!(
                "keyed archive value count {count} exceeds safety limit"
            ));
        }
        Ok(count)
    }

    fn value(
        &mut self,
        type_id: u8,
        fast_names: Option<&BTreeMap<u32, String>>,
        depth: usize,
        fast_string: bool,
    ) -> Result<KaValue, String> {
        if depth > MAX_DEPTH {
            return Err("SC2 keyed archives exceed the nesting limit".to_owned());
        }
        Ok(match type_id {
            0 => KaValue::Null,
            1 => KaValue::Bool(self.u8()? != 0),
            2 => KaValue::Signed(self.i32()? as i64),
            3 => KaValue::Float(self.f32()? as f64),
            4 if fast_string => KaValue::String(self.fast_name(fast_names)?),
            4 | 20 => KaValue::String(self.string()?),
            5 => {
                let count = self.u32()? as usize;
                let byte_length = count
                    .checked_mul(2)
                    .ok_or_else(|| "UTF-16 string length overflow".to_owned())?;
                let bytes = self.take(byte_length)?;
                let units = bytes
                    .chunks_exact(2)
                    .map(|pair| u16::from_le_bytes([pair[0], pair[1]]));
                KaValue::String(String::from_utf16_lossy(&units.collect::<Vec<_>>()))
            }
            6 => {
                let length = self.u32()? as usize;
                KaValue::Bytes(self.take(length)?.to_vec())
            }
            7 => KaValue::Unsigned(self.u32()? as u64),
            8 => {
                let expected_length = self.u32()? as usize;
                let start = self.position;
                let nested = self.archive(fast_names, depth + 1)?;
                let consumed = self.position - start;
                if consumed != expected_length {
                    return Err(format!(
                        "nested archive length mismatch: {consumed} != {expected_length}"
                    ));
                }
                nested
            }
            9 => KaValue::Signed(i64::from_le_bytes(self.take(8)?.try_into().unwrap())),
            10 => KaValue::Unsigned(u64::from_le_bytes(self.take(8)?.try_into().unwrap())),
            11..=14 => {
                let count = match type_id {
                    11 => 2,
                    12 => 3,
                    _ => 4,
                };
                KaValue::Array(
                    (0..count)
                        .map(|_| self.f32().map(|value| KaValue::Float(value as f64)))
                        .collect::<Result<Vec<_>, _>>()?,
                )
            }
            15 => self.float_array(9)?,
            16 => self.float_array(16)?,
            17 => KaValue::Bytes(self.take(4)?.to_vec()),
            18 => KaValue::String(self.fast_name(fast_names)?),
            19 => self.float_array(6)?,
            21 => KaValue::Float(f64::from_le_bytes(self.take(8)?.try_into().unwrap())),
            22 => KaValue::Signed(i8::from_le_bytes([self.u8()?]) as i64),
            23 => KaValue::Unsigned(self.u8()? as u64),
            24 => KaValue::Signed(i16::from_le_bytes(self.take(2)?.try_into().unwrap()) as i64),
            25 => KaValue::Unsigned(self.u16()? as u64),
            27 => {
                let count = self.count()?;
                let mut values = Vec::with_capacity(count);
                for _ in 0..count {
                    let item_type = self.u8()?;
                    values.push(self.value(item_type, fast_names, depth + 1, fast_string)?);
                }
                KaValue::Array(values)
            }
            29 => {
                let position = self.float_array(3)?;
                let scale = self.float_array(3)?;
                let rotation = self.float_array(4)?;
                KaValue::Map(BTreeMap::from([
                    ("position".to_owned(), position),
                    ("scale".to_owned(), scale),
                    ("rotation".to_owned(), rotation),
                ]))
            }
            other => return Err(format!("unsupported SC2 keyed archive value type {other}")),
        })
    }

    fn fast_name(&mut self, names: Option<&BTreeMap<u32, String>>) -> Result<String, String> {
        let index = self.u32()?;
        names
            .and_then(|table| table.get(&index))
            .cloned()
            .ok_or_else(|| format!("unknown SC2 fast-name id {index}"))
    }

    fn float_array(&mut self, count: usize) -> Result<KaValue, String> {
        let mut values = Vec::with_capacity(count);
        for _ in 0..count {
            values.push(KaValue::Float(self.f32()? as f64));
        }
        Ok(KaValue::Array(values))
    }
}

fn value_key(value: &KaValue) -> String {
    match value {
        KaValue::String(value) => value.clone(),
        KaValue::Signed(value) => value.to_string(),
        KaValue::Unsigned(value) => value.to_string(),
        KaValue::Bytes(value) => value.iter().map(|byte| format!("{byte:02x}")).collect(),
        KaValue::Bool(value) => value.to_string(),
        other => format!("{other:?}"),
    }
}

#[cfg(test)]
mod tests;
