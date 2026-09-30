package modem

import (
	"testing"
)

const pixel5Dumpsys = `SubscriptionManagerService:
Active modem count=2
Logical SIM slot sub id mapping:
  Logical SIM slot 1: subId=3
ICCID:
  slot 0: 
  slot 1: 896211218[****]

defaultSubId=3
defaultVoiceSubId=3
defaultDataSubId=3
activeDataSubId=3
defaultSmsSubId=3
areAllSubscriptionsLoaded=true
mUserIdToAvailableSubs={-10000=[3]}

mSimState[0]=ABSENT
mSimState[1]=LOADED

Active subscriptions:
  [SubscriptionInfoInternal: id=3 iccId=896211218[****] simSlotIndex=1 portIndex=0 isEmbedded=1 isRemovableEmbedded=0 carrierId=788 displayName=XL Axiata carrierName=XL Axiata isOpportunistic=0 groupUuid= groupOwner= displayNameSource=CARRIER iconTint=-15181124 number= dataRoaming=1 mcc=510 mnc=11 ehplmns= hplmns= cardString=890490320[****] cardId=0 nativeAccessRules= carrierConfigAccessRules= countryIso=id profileClass=2 type=LOCAL_SIM areUiccApplicationsEnabled=1 usageSetting=DEFAULT isEnhanced4GModeEnabled=-1 isVideoTelephonyEnabled=-1 isWifiCallingEnabled=1 isWifiCallingEnabledForRoaming=-1 wifiCallingMode=UNKNOWN wifiCallingModeForRoaming=UNKNOWN enabledMobileDataPolicies= imsi=510118101[****] rcsUceEnabled=0 crossSimCallingEnabled=0 rcsConfig= allowedNetworkTypesForReasons=user=588799 deviceToDeviceStatusSharingPreference=0 isVoImsOptInEnabled=0 deviceToDeviceStatusSharingContacts= numberFromCarrier= numberFromIms= userId=-10000 isSatelliteEnabled=0 satellite_attach_enabled_for_carrier=1 getOnlyNonTerrestrialNetwork=0 isGroupDisabled=false serviceCapabilities=7 transferStatus=0 satelliteEntitlementStatus=0 satelliteEntitlementPlmns= isSatelliteESOSSupported=0 isSatelliteProvisionedForNonIpDatagram=0 mSatelliteEntitlementBarredPlmnsList= mSatelliteEntitlementDataPlanForPlmn= mSatelliteEntitlementServicesForPlmn= mSatellitePlmnsDataServicePolicy= mSatellitePlmnsVoiceServicePolicy=]
`

const dualPhysicalSimDumpsys = `SubscriptionManagerService:
Active modem count=2
Logical SIM slot sub id mapping:
  Logical SIM slot 0: subId=1
  Logical SIM slot 1: subId=2

defaultSubId=1
defaultDataSubId=1
activeDataSubId=1

mSimState[0]=LOADED
mSimState[1]=LOADED

Active subscriptions:
  [SubscriptionInfoInternal: id=1 iccId=896200001[****] simSlotIndex=0 portIndex=0 isEmbedded=0 isRemovableEmbedded=0 carrierId=100 displayName=Telkomsel carrierName=Telkomsel isOpportunistic=0]
  [SubscriptionInfoInternal: id=2 iccId=896200002[****] simSlotIndex=1 portIndex=0 isEmbedded=0 isRemovableEmbedded=0 carrierId=200 displayName=Indosat Ooredoo carrierName=Indosat Ooredoo isOpportunistic=0]
`

func TestParseDumpsysIsubPixel5(t *testing.T) {
	status := parseDumpsysIsubInternal(pixel5Dumpsys, "", "")
	if status == nil {
		t.Fatalf("expected non-nil status")
	}

	if status.ActiveSlot != 1 {
		t.Errorf("expected ActiveSlot=1, got %d", status.ActiveSlot)
	}
	if status.ActiveSubID != 3 {
		t.Errorf("expected ActiveSubID=3, got %d", status.ActiveSubID)
	}
	if len(status.Slots) != 2 {
		t.Fatalf("expected 2 slots, got %d", len(status.Slots))
	}

	// Slot 0 check
	s0 := status.Slots[0]
	if s0.SlotIndex != 0 {
		t.Errorf("slot 0 index mismatch: %d", s0.SlotIndex)
	}
	if s0.State != "ABSENT" {
		t.Errorf("expected slot 0 state ABSENT, got %s", s0.State)
	}
	if s0.IsActive {
		t.Errorf("expected slot 0 IsActive=false")
	}
	if s0.IsESim {
		t.Errorf("expected slot 0 IsESim=false")
	}

	// Slot 1 check (Pixel 5 eSIM)
	s1 := status.Slots[1]
	if s1.SlotIndex != 1 {
		t.Errorf("slot 1 index mismatch: %d", s1.SlotIndex)
	}
	if s1.SubID != 3 {
		t.Errorf("expected slot 1 SubID=3, got %d", s1.SubID)
	}
	if s1.State != "LOADED" {
		t.Errorf("expected slot 1 state LOADED, got %s", s1.State)
	}
	if !s1.IsActive {
		t.Errorf("expected slot 1 IsActive=true")
	}
	if !s1.IsESim {
		t.Errorf("expected slot 1 IsESim=true")
	}
	if s1.Carrier != "XL Axiata" {
		t.Errorf("expected carrier 'XL Axiata', got '%s'", s1.Carrier)
	}
}

func TestParseDumpsysIsubDualPhysical(t *testing.T) {
	status := parseDumpsysIsubInternal(dualPhysicalSimDumpsys, "", "")
	if status == nil {
		t.Fatalf("expected non-nil status")
	}

	if status.ActiveSlot != 0 {
		t.Errorf("expected ActiveSlot=0, got %d", status.ActiveSlot)
	}
	if status.ActiveSubID != 1 {
		t.Errorf("expected ActiveSubID=1, got %d", status.ActiveSubID)
	}
	if len(status.Slots) != 2 {
		t.Fatalf("expected 2 slots, got %d", len(status.Slots))
	}

	// Slot 0 (Telkomsel pSIM)
	s0 := status.Slots[0]
	if s0.IsESim {
		t.Errorf("expected slot 0 IsESim=false")
	}
	if s0.Carrier != "Telkomsel" {
		t.Errorf("expected slot 0 carrier 'Telkomsel', got '%s'", s0.Carrier)
	}
	if !s0.IsActive {
		t.Errorf("expected slot 0 IsActive=true")
	}

	// Slot 1 (Indosat pSIM)
	s1 := status.Slots[1]
	if s1.IsESim {
		t.Errorf("expected slot 1 IsESim=false")
	}
	if s1.Carrier != "Indosat Ooredoo" {
		t.Errorf("expected slot 1 carrier 'Indosat Ooredoo', got '%s'", s1.Carrier)
	}
	if s1.IsActive {
		t.Errorf("expected slot 1 IsActive=false")
	}
}
