# simple-ad-placement-service
The server provides APIs for the advertisement placement service.
- Admin API：Create advertisement.
- Public API: Get the advertisements that meets the filter.

## Usage
```copy 
go run main.go
```

## APIs
### Admin API
**POST**  `/api/v1/ad`

Create advertisement.

#### Body Parameters
- `title` string  **_Required_**
  
  Title of advertisement
- `startAt` time  **_Required_**
  
  Active start time
- `endAt` time  **_Required_**
  
  Active end time
- `conditions` list of object  **_Required_**
  
  The advertisement is only active when meeting at least one of the following conditions.
  - `ageStart` integer
 
    The target's age must be greater than or equal to `ageStart`.
  - `ageEnd` integer
 
    The target's age must be greater than or equal to `ageStart`.
  - `gender` list of string
    
    "F" or "M".
    The target's gender must meet the list.
  - `country` list of string
 
    The target's location must be within the countries listed.
  
    The country code follows the ISO 3166-1 alpha-2 standard.
  - `platform` list of string

    The target's platform must be within the list.
  
    The element can be "android", "ios", or "web".

### Public API
**GET**  `/api/v1/ad`

Get the advertisements that meets the filter.

#### Query Parameters
- `offset` integer

  The parameter indicates from which data entry to start. It's 1-based.
- `limit` integer

  The parameter indicates the maximal number of returned advertisements.

  - Default: 5.
  - Range: 1~100.
- `age` integer

  The age of the target.
  - Range: 1~100.
- `gender` string
  
  The gender of the target.
- `country` string

  ISO 3166-1 alpha-2 code.
- `platform` string

  It can be "android", "ios", or "web".
